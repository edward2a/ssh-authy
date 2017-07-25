package main

import (
  "fmt"
  "log"
  //"net/http"
  "github.com/aws/aws-sdk-go/aws"
  //"github.com/aws/aws-sdk-go/aws/awserr"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/s3"
)

// Vars

var bucket = "test-bucket"
var project_base_path = "/projects"
var user_base_path = "/users"
var user_data_url = "169.254.169.254/latest/user-data"

func get_project_info() string {
  // Retrieve user-data
  m := "asdf"

  return m
}

func get_client() *s3.S3 {

  aws_session := session.Must(
    session.NewSessionWithOptions(
      session.Options{ SharedConfigState: session.SharedConfigEnable } ) )

  s3_client := s3.New(aws_session)
  return s3_client
}

func list_users(bkt string, path string, client *s3.S3) []string {

  resp, err := client.ListObjects(&s3.ListObjectsInput{
    Bucket: aws.String(bkt),
    Prefix: aws.String(path),
  } )
  if err != nil { log.Fatal("Failed to retrieve the user list") }

  user_list := make([]string, len(resp.Contents))
  for i := 0; i < len(resp.Contents); i++ {
    key := *resp.Contents[i].Key
    user_list[i] = key
  }

  return user_list
}

func get_keys(bkt string, path string, users []string, client *s3.S3) [][]byte {

  key_list := make([][]byte, len(users))

  for i := 0; i < len(users); i++ {
    user := users[i]
    resp, err := client.GetObject(&s3.GetObjectInput{
      Bucket: aws.String(bkt),
      Key: aws.String(user),
    })
    if err != nil { log.Printf("Failed to retrieve key for user: %s", user)
    } else {
      cl := *resp.ContentLength
      key_list[i] = make([]byte, int(cl))
      resp.Body.Read(key_list[i])
    }
    resp.Body.Close()
  }

  return key_list
}


func main() {

  s3_client := get_client()
  project_path := get_project_info()
  user_list := list_users(bucket, project_path, s3_client)
  creds_list := get_keys(bucket, user_base_path, user_list, s3_client)
  fmt.Println(creds_list)
}
