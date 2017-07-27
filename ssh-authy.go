package main

import (
  "fmt"
  "io/ioutil"
  "log"
  "net/http"
  "os"
  "strings"
  "time"
  "github.com/aws/aws-sdk-go/aws"
  //"github.com/aws/aws-sdk-go/aws/awserr"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/s3"
)

// Vars

var bucket = "my-test-bucket-asdf"
var project_base_path = "projects"
var user_base_path = "users"
var user_data_url = "http://169.254.169.254/latest/user-data"

var allowed_users = map[string]bool {
  "ec2-user": true,
  "ubuntu": true,
}

type platform struct {
  ProjectName string
  Environment string
}


func validate_input() {
  if len(os.Args) == 2 {
    if allowed_users[os.Args[1]] { ; } else { os.Exit(1) }
  } else {
    os.Exit(1)
  }
}

// Retrieve user-data (expecting project and env name here)
func get_project_info() platform {
  // temporarily return static data for testing
  // return platform{"myProject", "dev"}

  var prj string
  var env string

  client :=  &http.Client{ Timeout: time.Second * 5 }
  req, _ := http.NewRequest("GET", user_data_url, nil)

  resp, err := client.Do(req)
  if err != nil { log.Fatal("Failed to retrieve platform info") }
  defer resp.Body.Close()
  body, err := ioutil.ReadAll(resp.Body)
  if err != nil { log.Fatal("Failed parsing platform info") }

  vars := strings.Split(string(body), "\n")
  for i:=0; i < len(vars); i++ {
    if strings.HasPrefix(vars[i], "ProjectName=") {
      prj = strings.TrimPrefix(vars[i], "ProjectName=")
    } else if strings.HasPrefix(vars[i], "Environment=") {
      env = strings.TrimPrefix(vars[i], "Environment=")
    }
  }

  return platform{prj, env}
}

// Instantiate an S3 client
func get_client() *s3.S3 {

  // Need to add region!
  aws_session := session.Must(
    session.NewSessionWithOptions(
      session.Options{ SharedConfigState: session.SharedConfigEnable } ) )

  s3_client := s3.New(aws_session)
  return s3_client
}

// Get list of users for a project/env
func list_users(bkt string, path platform, client *s3.S3) []string {

  prefix := project_base_path + "/" + path.ProjectName + "/" + path.Environment + "/"

  resp, err := client.ListObjects(&s3.ListObjectsInput{
    Bucket: aws.String(bkt),
    Prefix: aws.String(prefix),
  } )
  if err != nil { log.Fatal(err.Error()) }

  user_list := make([]string, len(resp.Contents))
  for i := 0; i < len(resp.Contents); i++ {
    key := strings.Split(*resp.Contents[i].Key, "/")
    user_list[i] = key[len(key)-1]
  }

  return user_list
}

// Get SSH keys for each user provided in `users`
func get_keys(bkt string, path string, users []string, client *s3.S3) [][]byte {

  key_list := make([][]byte, len(users))

  for i := 0; i < len(users); i++ {
    user := user_base_path + "/" + users[i]
    resp, err := client.GetObject(&s3.GetObjectInput{
      Bucket: aws.String(bkt),
      Key: aws.String(user),
    })
    if err != nil { log.Printf("Failed to retrieve key for user: %s", user)
    } else {
      cl := *resp.ContentLength
      key_list[i] = make([]byte, int(cl))
      resp.Body.Read(key_list[i])
      resp.Body.Close()
    }
  }

  return key_list
}


func main() {
  validate_input()
  s3_client := get_client()
  project_data := get_project_info()
  user_list := list_users(bucket, project_data, s3_client)
  creds_list := get_keys(bucket, user_base_path, user_list, s3_client)
  keys_list := make([]string, len(creds_list))

  // Array of byte-arrrays to array of strings
  for i:=0; i<len(creds_list); i++ {
    keys_list[i] = string(creds_list[i])
  }

  // Output as per SSH's expected format
  fmt.Println(strings.Join(keys_list, "\n"))
}
