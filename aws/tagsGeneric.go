package aws

import (
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
)




// DiffTags takes our tags locally and the ones remotely and returns
// the set of tags that must be created, and the set of tags that must
// be destroyed.
func DiffTagsGeneric(oldTags, newTags map[string]interface{}) (map[string]*string, map[string]*string) {
	// First, we're creating everything we have
	create := make(map[string]*string)
	for k, v := range newTags {
		create[k] = aws.String(v.(string))
	}

	// Build the map of what to remove
	remove := make(map[string]*string)
	for k, v := range oldTags {
		old, ok := create[k]
		if !ok || old != aws.String(v.(string)) {
			// Delete it!
			remove[k] = aws.String(v.(string))
		}
	}

	return create, remove
}

// TagsFromMap returns the tags for the given map of data.
func TagsFromMapGeneric(m map[string]interface{}) map[string]*string {
	result := make(map[string]*string)
	for k, v := range m {
		if !TagIgnoredGeneric(k) {
			result[k] = aws.String(v.(string))
		}
	}

	return result
}

// TagsToMap turns the tags into a map.
func TagsToMapGeneric(ts map[string]*string) map[string]string {
	result := make(map[string]string)
	for k, v := range ts {
		if !TagIgnoredGeneric(k) {
			result[k] = aws.StringValue(v)
		}
	}

	return result
}

// compare a tag against a list of strings and checks if it should
// be ignored or not
func TagIgnoredGeneric(k string) bool {
	filter := []string{"^aws:"}
	for _, v := range filter {
		log.Printf("[DEBUG] Matching %v with %v\n", v, k)
		if r, _ := regexp.MatchString(v, k); r == true {
			log.Printf("[DEBUG] Found AWS specific tag %s, ignoring.\n", k)
			return true
		}
	}
	return false
}
