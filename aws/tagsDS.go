package aws

import (
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/directoryservice"
	"github.com/hashicorp/terraform/helper/schema"
)

// SetTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func SetTagsDS(conn *directoryservice.DirectoryService, d *schema.ResourceData, resourceId string) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := DiffTagsDS(TagsFromMapDS(o), TagsFromMapDS(n))

		// Set tags
		if len(remove) > 0 {
			log.Printf("[DEBUG] Removing tags: %s", remove)
			k := make([]*string, len(remove), len(remove))
			for i, t := range remove {
				k[i] = t.Key
			}

			_, err := conn.RemoveTagsFromResource(&directoryservice.RemoveTagsFromResourceInput{
				ResourceId: aws.String(resourceId),
				TagKeys:    k,
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			log.Printf("[DEBUG] Creating tags: %s", create)
			_, err := conn.AddTagsToResource(&directoryservice.AddTagsToResourceInput{
				ResourceId: aws.String(resourceId),
				Tags:       create,
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
func DiffTagsDS(oldTags, newTags []*directoryservice.Tag) ([]*directoryservice.Tag, []*directoryservice.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []*directoryservice.Tag
	for _, t := range oldTags {
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return TagsFromMapDS(create), remove
}

// TagsFromMap returns the tags for the given map of data.
func TagsFromMapDS(m map[string]interface{}) []*directoryservice.Tag {
	result := make([]*directoryservice.Tag, 0, len(m))
	for k, v := range m {
		t := &directoryservice.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		}
		if !TagIgnoredDS(t) {
			result = append(result, t)
		}
	}

	return result
}

// TagsToMap turns the list of tags into a map.
func TagsToMapDS(ts []*directoryservice.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		if !TagIgnoredDS(t) {
			result[*t.Key] = *t.Value
		}
	}

	return result
}

// compare a tag against a list of strings and checks if it should
// be ignored or not
func TagIgnoredDS(t *directoryservice.Tag) bool {
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
