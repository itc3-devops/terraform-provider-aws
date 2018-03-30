package aws

import (
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/hashicorp/terraform/helper/schema"
)

// SetTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func SetTagsKMS(conn *kms.KMS, d *schema.ResourceData, keyId string) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := DiffTagsKMS(TagsFromMapKMS(o), TagsFromMapKMS(n))

		// Set tags
		if len(remove) > 0 {
			log.Printf("[DEBUG] Removing tags: %#v", remove)
			k := make([]*string, len(remove), len(remove))
			for i, t := range remove {
				k[i] = t.TagKey
			}

			_, err := conn.UntagResource(&kms.UntagResourceInput{
				KeyId:   aws.String(keyId),
				TagKeys: k,
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			log.Printf("[DEBUG] Creating tags: %#v", create)
			_, err := conn.TagResource(&kms.TagResourceInput{
				KeyId: aws.String(keyId),
				Tags:  create,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// DiffTags takes our tags locally and the ones remotely and returns
// the set of tags that must be created, and the set of tags that must
// be destroyed.
func DiffTagsKMS(oldTags, newTags []*kms.Tag) ([]*kms.Tag, []*kms.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[aws.StringValue(t.TagKey)] = aws.StringValue(t.TagValue)
	}

	// Build the list of what to remove
	var remove []*kms.Tag
	for _, t := range oldTags {
		old, ok := create[aws.StringValue(t.TagKey)]
		if !ok || old != aws.StringValue(t.TagValue) {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return TagsFromMapKMS(create), remove
}

// TagsFromMap returns the tags for the given map of data.
func TagsFromMapKMS(m map[string]interface{}) []*kms.Tag {
	result := make([]*kms.Tag, 0, len(m))
	for k, v := range m {
		t := &kms.Tag{
			TagKey:   aws.String(k),
			TagValue: aws.String(v.(string)),
		}
		if !TagIgnoredKMS(t) {
			result = append(result, t)
		}
	}

	return result
}

// TagsToMap turns the list of tags into a map.
func TagsToMapKMS(ts []*kms.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		if !TagIgnoredKMS(t) {
			result[aws.StringValue(t.TagKey)] = aws.StringValue(t.TagValue)
		}
	}

	return result
}

// compare a tag against a list of strings and checks if it should
// be ignored or not
func TagIgnoredKMS(t *kms.Tag) bool {
	filter := []string{"^aws:"}
	for _, v := range filter {
		log.Printf("[DEBUG] Matching %v with %v\n", v, *t.TagKey)
		if r, _ := regexp.MatchString(v, *t.TagKey); r == true {
			log.Printf("[DEBUG] Found AWS specific tag %s (val: %s), ignoring.\n", *t.TagKey, *t.TagValue)
			return true
		}
	}
	return false
}
