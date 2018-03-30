package aws

import (
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
)

// DiffTags takes our tags locally and the ones remotely and returns
// the set of tags that must be created, and the set of tags that must
// be destroyed.
func DiffTagsBeanstalk(oldTags, newTags []*elasticbeanstalk.Tag) ([]*elasticbeanstalk.Tag, []*string) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []*string
	for _, t := range oldTags {
		if _, ok := create[*t.Key]; !ok {
			// Delete it!
			remove = append(remove, t.Key)
		}
	}

	return TagsFromMapBeanstalk(create), remove
}

// TagsFromMap returns the tags for the given map of data.
func TagsFromMapBeanstalk(m map[string]interface{}) []*elasticbeanstalk.Tag {
	var result []*elasticbeanstalk.Tag
	for k, v := range m {
		t := &elasticbeanstalk.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		}
		if !TagIgnoredBeanstalk(t) {
			result = append(result, t)
		}
	}

	return result
}

// TagsToMap turns the list of tags into a map.
func TagsToMapBeanstalk(ts []*elasticbeanstalk.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		if !TagIgnoredBeanstalk(t) {
			result[*t.Key] = *t.Value
		}
	}

	return result
}

// compare a tag against a list of strings and checks if it should
// be ignored or not
func TagIgnoredBeanstalk(t *elasticbeanstalk.Tag) bool {
	filter := []string{"^aws:", "^elasticbeanstalk:", "Name"}
	for _, v := range filter {
		log.Printf("[DEBUG] Matching %v with %v\n", v, *t.Key)
		if r, _ := regexp.MatchString(v, *t.Key); r == true {
			log.Printf("[DEBUG] Found AWS specific tag %s (val: %s), ignoring.\n", *t.Key, *t.Value)
			return true
		}
	}
	return false
}
