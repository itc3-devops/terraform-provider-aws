package aws

import (
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/hashicorp/terraform/helper/schema"
)

// SetTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func SetTagsCloudtrail(conn *cloudtrail.CloudTrail, d *schema.ResourceData) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := DiffTagsCloudtrail(TagsFromMapCloudtrail(o), TagsFromMapCloudtrail(n))

		// Set tags
		if len(remove) > 0 {
			input := cloudtrail.RemoveTagsInput{
				ResourceId: aws.String(d.Get("arn").(string)),
				TagsList:   remove,
			}
			log.Printf("[DEBUG] Removing CloudTrail tags: %s", input)
			_, err := conn.RemoveTags(&input)
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			input := cloudtrail.AddTagsInput{
				ResourceId: aws.String(d.Get("arn").(string)),
				TagsList:   create,
			}
			log.Printf("[DEBUG] Adding CloudTrail tags: %s", input)
			_, err := conn.AddTags(&input)
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
func DiffTagsCloudtrail(oldTags, newTags []*cloudtrail.Tag) ([]*cloudtrail.Tag, []*cloudtrail.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []*cloudtrail.Tag
	for _, t := range oldTags {
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return TagsFromMapCloudtrail(create), remove
}

// TagsFromMap returns the tags for the given map of data.
func TagsFromMapCloudtrail(m map[string]interface{}) []*cloudtrail.Tag {
	var result []*cloudtrail.Tag
	for k, v := range m {
		t := &cloudtrail.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		}
		if !TagIgnoredCloudtrail(t) {
			result = append(result, t)
		}
	}

	return result
}

// TagsToMap turns the list of tags into a map.
func TagsToMapCloudtrail(ts []*cloudtrail.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		if !TagIgnoredCloudtrail(t) {
			result[*t.Key] = *t.Value
		}
	}

	return result
}

// compare a tag against a list of strings and checks if it should
// be ignored or not
func TagIgnoredCloudtrail(t *cloudtrail.Tag) bool {
	filter := []string{"^aws:"}
	for _, v := range filter {
		log.Printf("[DEBUG] Matching %v with %v\n", v, *t.Key)
		if r, _ := regexp.MatchString(v, *t.Key); r == true {
			log.Printf("[DEBUG] Found AWS specific tag %s (val: %s), ignoring.\n", *t.Key, *t.Value)
			return true
		}
	}
	return false
}
