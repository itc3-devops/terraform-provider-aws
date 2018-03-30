package aws

import (
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	elasticsearch "github.com/aws/aws-sdk-go/service/elasticsearchservice"
	"github.com/hashicorp/terraform/helper/schema"
)

// SetTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func SetTagsElasticsearchService(conn *elasticsearch.ElasticsearchService, d *schema.ResourceData, arn string) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := DiffTagsElasticsearchService(TagsFromMapElasticsearchService(o), TagsFromMapElasticsearchService(n))

		// Set tags
		if len(remove) > 0 {
			log.Printf("[DEBUG] Removing tags: %#v", remove)
			k := make([]*string, 0, len(remove))
			for _, t := range remove {
				k = append(k, t.Key)
			}
			_, err := conn.RemoveTags(&elasticsearch.RemoveTagsInput{
				ARN:     aws.String(arn),
				TagKeys: k,
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			log.Printf("[DEBUG] Creating tags: %#v", create)
			_, err := conn.AddTags(&elasticsearch.AddTagsInput{
				ARN:     aws.String(arn),
				TagList: create,
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
func DiffTagsElasticsearchService(oldTags, newTags []*elasticsearch.Tag) ([]*elasticsearch.Tag, []*elasticsearch.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []*elasticsearch.Tag
	for _, t := range oldTags {
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return TagsFromMapElasticsearchService(create), remove
}

// TagsFromMap returns the tags for the given map of data.
func TagsFromMapElasticsearchService(m map[string]interface{}) []*elasticsearch.Tag {
	var result []*elasticsearch.Tag
	for k, v := range m {
		t := &elasticsearch.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		}
		if !TagIgnoredElasticsearchService(t) {
			result = append(result, t)
		}
	}

	return result
}

// TagsToMap turns the list of tags into a map.
func TagsToMapElasticsearchService(ts []*elasticsearch.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		if !TagIgnoredElasticsearchService(t) {
			result[*t.Key] = *t.Value
		}
	}

	return result
}

// compare a tag against a list of strings and checks if it should
// be ignored or not
func TagIgnoredElasticsearchService(t *elasticsearch.Tag) bool {
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
