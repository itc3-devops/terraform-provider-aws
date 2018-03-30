package aws

import (
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

// TagsSchema returns the schema to use for tags.
//
func TagsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
	}
}

func TagsSchemaComputed() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		Computed: true,
	}
}

func SetElbV2Tags(conn *elbv2.ELBV2, d *schema.ResourceData) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := DiffElbV2Tags(TagsFromMapELBv2(o), TagsFromMapELBv2(n))

		// Set tags
		if len(remove) > 0 {
			var tagKeys []*string
			for _, tag := range remove {
				tagKeys = append(tagKeys, tag.Key)
			}
			log.Printf("[DEBUG] Removing tags: %#v from %s", remove, d.Id())
			_, err := conn.RemoveTags(&elbv2.RemoveTagsInput{
				ResourceArns: []*string{aws.String(d.Id())},
				TagKeys:      tagKeys,
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			log.Printf("[DEBUG] Creating tags: %s for %s", create, d.Id())
			_, err := conn.AddTags(&elbv2.AddTagsInput{
				ResourceArns: []*string{aws.String(d.Id())},
				Tags:         create,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func SetVolumeTags(conn *ec2.EC2, d *schema.ResourceData) error {
	if d.HasChange("volume_tags") {
		oraw, nraw := d.GetChange("volume_tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := DiffTags(TagsFromMap(o), TagsFromMap(n))

		volumeIds, err := getAwsInstanceVolumeIds(conn, d)
		if err != nil {
			return err
		}

		if len(remove) > 0 {
			err := resource.Retry(2*time.Minute, func() *resource.RetryError {
				log.Printf("[DEBUG] Removing volume tags: %#v from %s", remove, d.Id())
				_, err := conn.DeleteTags(&ec2.DeleteTagsInput{
					Resources: volumeIds,
					Tags:      remove,
				})
				if err != nil {
					ec2err, ok := err.(awserr.Error)
					if ok && strings.Contains(ec2err.Code(), ".NotFound") {
						return resource.RetryableError(err) // retry
					}
					return resource.NonRetryableError(err)
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			err := resource.Retry(2*time.Minute, func() *resource.RetryError {
				log.Printf("[DEBUG] Creating vol tags: %s for %s", create, d.Id())
				_, err := conn.CreateTags(&ec2.CreateTagsInput{
					Resources: volumeIds,
					Tags:      create,
				})
				if err != nil {
					ec2err, ok := err.(awserr.Error)
					if ok && strings.Contains(ec2err.Code(), ".NotFound") {
						return resource.RetryableError(err) // retry
					}
					return resource.NonRetryableError(err)
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// SetTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func SetTags(conn *ec2.EC2, d *schema.ResourceData) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := DiffTags(TagsFromMap(o), TagsFromMap(n))

		// Set tags
		if len(remove) > 0 {
			err := resource.Retry(5*time.Minute, func() *resource.RetryError {
				log.Printf("[DEBUG] Removing tags: %#v from %s", remove, d.Id())
				_, err := conn.DeleteTags(&ec2.DeleteTagsInput{
					Resources: []*string{aws.String(d.Id())},
					Tags:      remove,
				})
				if err != nil {
					ec2err, ok := err.(awserr.Error)
					if ok && strings.Contains(ec2err.Code(), ".NotFound") {
						return resource.RetryableError(err) // retry
					}
					return resource.NonRetryableError(err)
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			err := resource.Retry(5*time.Minute, func() *resource.RetryError {
				log.Printf("[DEBUG] Creating tags: %s for %s", create, d.Id())
				_, err := conn.CreateTags(&ec2.CreateTagsInput{
					Resources: []*string{aws.String(d.Id())},
					Tags:      create,
				})
				if err != nil {
					ec2err, ok := err.(awserr.Error)
					if ok && strings.Contains(ec2err.Code(), ".NotFound") {
						return resource.RetryableError(err) // retry
					}
					return resource.NonRetryableError(err)
				}
				return nil
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
func DiffTags(oldTags, newTags []*ec2.Tag) ([]*ec2.Tag, []*ec2.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []*ec2.Tag
	for _, t := range oldTags {
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			remove = append(remove, t)
		}
	}

	return TagsFromMap(create), remove
}

// TagsFromMap returns the tags for the given map of data.
func TagsFromMap(m map[string]interface{}) []*ec2.Tag {
	result := make([]*ec2.Tag, 0, len(m))
	for k, v := range m {
		t := &ec2.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		}
		if !TagIgnored(t) {
			result = append(result, t)
		}
	}

	return result
}

// TagsToMap turns the list of tags into a map.
func TagsToMap(ts []*ec2.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		if !TagIgnored(t) {
			result[*t.Key] = *t.Value
		}
	}

	return result
}

func DiffElbV2Tags(oldTags, newTags []*elbv2.Tag) ([]*elbv2.Tag, []*elbv2.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []*elbv2.Tag
	for _, t := range oldTags {
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return TagsFromMapELBv2(create), remove
}

// TagsToMapELBv2 turns the list of tags into a map.
func TagsToMapELBv2(ts []*elbv2.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		if !TagIgnoredELBv2(t) {
			result[*t.Key] = *t.Value
		}
	}

	return result
}

// TagsFromMapELBv2 returns the tags for the given map of data.
func TagsFromMapELBv2(m map[string]interface{}) []*elbv2.Tag {
	var result []*elbv2.Tag
	for k, v := range m {
		t := &elbv2.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		}
		if !TagIgnoredELBv2(t) {
			result = append(result, t)
		}
	}

	return result
}

// TagIgnored compares a tag against a list of strings and checks if it should
// be ignored or not
func TagIgnored(t *ec2.Tag) bool {
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

// and for ELBv2 as well
func TagIgnoredELBv2(t *elbv2.Tag) bool {
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

// TagsToMapDynamoDb turns the list of tags into a map for dynamoDB
func TagsToMapDynamoDb(ts []*dynamodb.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		result[*t.Key] = *t.Value
	}
	return result
}

// TagsFromMapDynamoDb returns the tags for a given map
func TagsFromMapDynamoDb(m map[string]interface{}) []*dynamodb.Tag {
	result := make([]*dynamodb.Tag, 0, len(m))
	for k, v := range m {
		t := &dynamodb.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		}
		result = append(result, t)
	}
	return result
}

// SetTagsDynamoDb is a helper to set the tags for a dynamoDB resource
// This is needed because dynamodb requires a completely different set and delete
// method from the ec2 tag resource handling. Also the `UntagResource` method
// for dynamoDB only requires a list of tag keys, instead of the full map of keys.
func SetTagsDynamoDb(conn *dynamodb.DynamoDB, d *schema.ResourceData) error {
	arn := d.Get("arn").(string)
	oraw, nraw := d.GetChange("tags")
	o := oraw.(map[string]interface{})
	n := nraw.(map[string]interface{})
	create, remove := DiffTagsDynamoDb(TagsFromMapDynamoDb(o), TagsFromMapDynamoDb(n))

	// Set tags
	if len(remove) > 0 {
		err := resource.Retry(2*time.Minute, func() *resource.RetryError {
			log.Printf("[DEBUG] Removing tags: %#v from %s", remove, d.Id())
			_, err := conn.UntagResource(&dynamodb.UntagResourceInput{
				ResourceArn: aws.String(arn),
				TagKeys:     remove,
			})
			if err != nil {
				if isAWSErr(err, dynamodb.ErrCodeResourceNotFoundException, "") {
					return resource.RetryableError(err)
				}
				return resource.NonRetryableError(err)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	if len(create) > 0 {
		err := resource.Retry(2*time.Minute, func() *resource.RetryError {
			log.Printf("[DEBUG] Creating tags: %s for %s", create, d.Id())
			_, err := conn.TagResource(&dynamodb.TagResourceInput{
				ResourceArn: aws.String(arn),
				Tags:        create,
			})
			if err != nil {
				if isAWSErr(err, dynamodb.ErrCodeResourceNotFoundException, "") {
					return resource.RetryableError(err)
				}
				return resource.NonRetryableError(err)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// DiffTagsDynamoDb takes a local set of dynamodb tags and the ones found remotely
// and returns the set of tags that must be created as a map, and returns a list of tag keys
// that must be destroyed.
func DiffTagsDynamoDb(oldTags, newTags []*dynamodb.Tag) ([]*dynamodb.Tag, []*string) {
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	var remove []*string
	for _, t := range oldTags {
		// Verify the old tag is not a tag we're currently attempting to create
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			remove = append(remove, t.Key)
		}
	}
	return TagsFromMapDynamoDb(create), remove
}
