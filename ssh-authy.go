package main

import (
  "encoding/json"
  "fmt"
  "io/ioutil"
  "log"
  "log/syslog"
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
var instance_id_url = "http://169.254.169.254/latest/dynamic/instance-identity/document"
var syslog_writer *syslog.Writer
var proj_id_field = "ProjectName="
var env_id_field = "Environment="

var allowed_users = map[string]bool {
  "ec2-user": true,
  "ubuntu": true,
}

type platform struct {
  ProjectName string
  Environment string
  Region string
}

func config_logger() {
  syslog_writer, err := syslog.New(syslog.LOG_AUTH, "ssh-authy")
  if err != nil { os.Exit(2) }
  log.SetOutput(syslog_writer)
  log.SetFlags(0)
}

func validate_input() {
  if len(os.Args) == 2 {
    if allowed_users[os.Args[1]] { ; } else { log.Fatal("Username not allowed for lookup") }
  } else {
    log.Fatal("Missing username")
  }
}

// Retrieve user-data (expecting project and env name here)
func get_project_info() platform {
  // temporarily return static data for testing
  // return platform{"myProject", "dev"}

  var prj string
  var env string
  var idoc map[string]interface{}

  client :=  &http.Client{ Timeout: time.Second * 2 }
  udata_req, _ := http.NewRequest("GET", user_data_url, nil)
  idoc_req, _ := http.NewRequest("GET", instance_id_url, nil)

  // user-data
  udata_resp, err := client.Do(udata_req)
  if err != nil { log.Fatal("Failed to retrieve platform info") }
  defer udata_resp.Body.Close()
  body, err := ioutil.ReadAll(udata_resp.Body)
  if err != nil { log.Fatal("Failed parsing platform info") }

  vars := strings.Split(string(body), "\n")
  for i:=0; i < len(vars); i++ {
    if strings.HasPrefix(vars[i], proj_id_field) {
      prj = strings.TrimPrefix(vars[i], proj_id_field)
    } else if strings.HasPrefix(vars[i], proj_id_field) {
      env = strings.TrimPrefix(vars[i], proj_id_field)
    }
  }

  // instance identity doc
  idoc_resp, err := client.Do(idoc_req)
  if err != nil { log.Fatal("Failed to retrieve platform identity document") }
  defer idoc_resp.Body.Close()
  idoc_dec := json.NewDecoder(idoc_resp.Body)
  idoc_dec.Decode(&idoc)

  return platform{prj, env, idoc["region"].(string)}
}

// Instantiate an S3 client
func get_client(region string) *s3.S3 {

  // Need to add region!
  aws_session := session.Must(
    session.NewSession(&aws.Config{
      Region: aws.String(region) } ) )

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
  config_logger()
  validate_input()
  project_data := get_project_info()
  s3_client := get_client(project_data.Region)
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
