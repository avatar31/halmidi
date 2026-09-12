# halmidi

### Installation

Create Data directory
```sh
mkdir -p /data/halmidi
chmod 777 /data/halmidi
```

Add below content in the file `/etc/halmidi/halmidi.conf`
```bash
[general]
basepath = /data/halmidi

[logging]
path = /tmp/halmidi
log-level = info
```

### S3 Notes
- https://docs.aws.amazon.com/AmazonS3/latest/userguide/Welcome.html

### Supported commands

#### Bucket CRUD operations
- `aws --no-sign-request --endpoint-url http://localhost:9051 s3 ls`
- `aws s3 ls --no-sign-request --debug`
- `aws --endpoint-url http://localhost:9051 s3api create-bucket --bucket my-bucket1 --object-lock-enabled-for-bucket`
- `aws --endpoint-url http://localhost:9051 s3 cp .\latlog_clat.1.log s3://my-bucket1`
- `aws --endpoint-url http://localhost:9051 s3 ls s3://my-bucket1`
- `aws s3api list-objects-v2 --bucket my-bucket1`
- `aws --endpoint-url http://localhost:9051 s3 rm s3://my-bucket1/latlog_clat.1.log`
- `aws --endpoint-url http://localhost:9051 s3api delete-object --bucket my-bucket1 --key latlog_clat.2.log --bypass-governance-retention`
- `aws s3api delete-object --bucket bucket5 --key xyz.txt --version-id a5e3b0ad-046f-47dc-b612-e2440026c015`


#### Versioning
- Get
```sh
aws s3api get-bucket-versioning --bucket bucket5
aws s3api list-object-versions --bucket bucket5 --prefix xyz.txt
```

- Put
```sh
aws s3api put-bucket-versioning --bucket bucket5 --versioning-configuration Status=Enabled
```

#### Object Locking
- Get Object Locking
```sh
aws --endpoint-url http://localhost:9051 s3api get-object-lock-configuration --bucket my-bucket
```

- Update Object Locking 
```sh
aws --no-sign-request \
  --endpoint-url http://localhost:9051 s3api put-object-lock-configuration \
  --bucket my-bucket \
  --object-lock-configuration '{
    "ObjectLockEnabled": "Enabled",
    "Rule": {
      "DefaultRetention": {
        "Mode": "COMPLIANCE",
        "Years": 1
      }
    }
  }'
```

#### Legal Hold
- Get
```sh
aws --no-sign-request --endpoint-url http://localhost:9051 s3api get-object-legal-hold --bucket my-bucket --key latlog_clat.1.log
```

- Put
```sh
aws --endpoint-url http://localhost:9051 s3api put-object-legal-hold --bucket my-bucket1 --key latlog_clat.2.log --legal-hold Status=ON
```

#### Tagging
- Get
```sh
aws --endpoint-url http://localhost:9051 s3api get-bucket-tagging --bucket my-bucket1
aws --endpoint-url http://localhost:9051 s3api get-object-tagging --bucket my-bucket1 --key latlog_clat.1.log
```

- Put
```sh
aws --endpoint-url http://localhost:9051 s3api put-bucket-tagging --bucket my-bucket1 --tagging 'TagSet=[{Key=Environment,Value=Test},{Key=Owner,Value=Alice}]'
aws --endpoint-url http://localhost:9051 s3api put-object-tagging --bucket my-bucket1 --key latlog_clat.1.log --tagging 'TagSet=[{Key=Environment,Value=Test},{Key=Owner,Value=Alice}]'
```

- Delete
```sh
aws --endpoint-url http://localhost:9051 s3api delete-bucket-tagging --bucket my-bucket1
aws --endpoint-url http://localhost:9051 s3api delete-object-tagging --bucket my-bucket1 --key latlog_clat.1.log
```

#### Lifecycle

```sh
aws s3api get-bucket-lifecycle-configuration --bucket abc
aws s3api put-bucket-lifecycle --bucket abc --lifecycle-configuration file://assets/lifecycle.json
aws s3api delete-bucket-lifecycle --bucket abc
```

#### IAM Policies
```sh
aws iam create-policy --policy-name policy4 --description "Allows listing S3 bucket" --path /department/security/ --policy-document file://assets/policy.json --tags Key=Department,Value=Finance Key=Env,Value=Prod
aws iam get-policy --policy-arn arn:aws:iam::default:policy/department/security/policy2
aws iam list-policies
aws iam delete-policy --policy-arn arn:aws:iam::default:policy/policy1
aws iam list-policy-tags --policy-arn arn:aws:iam::default:policy/department/security/policy4
aws iam tag-policy --policy-arn arn:aws:iam::default:policy/department/security/policy4 --tags Key=Status,Value=Temp Key=Env,Value=Dev
aws iam untag-policy --policy-arn arn:aws:iam::default:policy/department/security/policy4 --tag-keys Status
aws iam list-entities-for-policy --policy-arn arn:aws:iam::default:policy/department/security/policy4
```

#### IAM Policies Versions
```sh
aws iam create-policy --policy-name policy3 --description "Allows listing S3 bucket" --path /department/security/ --policy-document file://assets/policy.json
aws iam list-policy-versions --policy-arn arn:aws:iam::default:policy/department/security/policy3
aws iam create-policy-version --policy-arn arn:aws:iam::default:policy/department/security/policy3 --policy-document file://assets/policy_v1.json --set-as-default
aws iam delete-policy-version --policy-arn arn:aws:iam::default:policy/department/security/policy3 --version-id v3
aws iam get-policy-version --policy-arn arn:aws:iam::default:policy/department/security/policy3 --version-id v3
aws iam set-default-policy-version --policy-arn arn:aws:iam::default:policy/department/security/policy3 --version-id v3
```

#### IAM Groups
```sh
aws iam create-group --group-name Developers --path /teams/
aws iam list-groups
aws iam get-group --group-name Developers
aws iam update-group --group-name Developers --new-path /
aws iam add-user-to-group --group-name Developers --user-name sachin
aws iam remove-user-from-group --group-name Developers --user-name sachin
aws iam attach-group-policy --group-name Developers --policy-arn arn:aws:iam::default:policy/department/security/policy4
aws iam list-attached-group-policies --group-name Developers
aws iam detach-group-policy --group-name Developers --policy-arn arn:aws:iam::default:policy/department/security/policy4
aws iam delete-group --group-name Developers
aws iam put-group-policy --group-name Developers --policy-name inline-policy1 --policy-document file://assets/policy_v1.json
aws iam get-group-policy --group-name Developers --policy-name inline-policy1
aws iam delete-group-policy --group-name Developers --policy-name inline-policy1
```

#### IAM Users

```sh
aws iam create-user --user-name sachin --tags Key=Department,Value=Finance Key=Env,Value=Prod
aws iam delete-user --user-name sachin
aws iam get-user --user-name sachin
aws iam list-users
aws iam list-groups-for-user --user-name sachin
aws iam list-user-tags --user-name sachin
aws iam tag-user --user-name sachin --tags Key=Status,Value=Temp Key=Env,Value=Dev
aws iam untag-user --user-name sachin --tag-keys Status
aws iam attach-user-policy --user-name sachin --policy-arn arn:aws:iam::default:policy/department/security/policy4
aws iam list-attached-user-policies --user-name sachin
aws iam detach-user-policy --user-name sachin --policy-arn arn:aws:iam::default:policy/department/security/policy4
aws iam put-user-policy --user-name sachin --policy-name inline-policy1 --policy-document file://assets/policy_v1.json
aws iam get-user-policy --user-name sachin --policy-name inline-policy1
aws iam delete-user-policy --user-name sachin --policy-name inline-policy1
```

#### IAM User Access Keys

```sh
aws iam create-access-key --user-name sachin
aws iam list-access-keys --user-name sachin
aws iam delete-access-key --user-name sachin --access-key-id P3S3CMNTNYXMARVIEYS
aws iam update-access-key --user-name sachin --access-key-id P3SO6EYQVZAG55JFB23 --status Inactive
aws iam get-access-key-last-used --access-key-id AFAZWFIKR34Y4MPU53X

# 10.18.104.107
{
    "AccessKey": {
      "userName":"sachin",
      "accessKeyId":"AFADBV7LQYOLM3F3TY2",
      "secretAccessKey":"iCLn2SwY5ZhspQrRh641aYnZmJxMAe1sIdegB0+Atmg=",
      "status":"Active",
      "createDate":"2025-09-18T17:03:37Z"
    }
}

# 10.18.104.120

{
    "AccessKey": {
      "userName":"sachin",
      "accessKeyId":"AFA3WLV23VIHNC7VBEO",
      "secretAccessKey":"JuZEQt4AikFWbe9LSXn4dPe5DPnv0O1jJrshzkmcqSk=",
      "status":"Active",
      "createDate":"2025-09-18T17:03:37Z"
    }
}

# 10.18.104.157

{
    "AccessKey": {
      "userName":"sachin",
      "accessKeyId":"AFAQDA2AI7D5B55YUAQ",
      "secretAccessKey":"s5IqiD5vfn+hQlgNne199fDBH+sCQ0zKEYABsBHMdVo=",
      "status":"Active",
      "createDate":"2025-09-18T17:03:37Z"
    }
}


```

### API's
#### IAM Users
```sh
curl -X POST -i http://localhost:9501/api/v1/users -d '{"userName": "sachin"}'
curl -X POST -i http://localhost:9511/api/v1/users/sachin/accesskey
```

### Benchmark
```sh
warp mixed --host=localhost:9051 --access-key=AFAHO5DVADPSOZHGWBU --secret-key=wFki9O4Rsh7ES+87QEMlOaScHZ6LO5V/1NwnyPRiSAo= --obj.size=1M --bucket=warp-stat3
warp mixed --host=localhost:9051 --access-key=AFAHO5DVADPSOZHGWBU --secret-key=wFki9O4Rsh7ES+87QEMlOaScHZ6LO5V/1NwnyPRiSAo= --obj.size=1M --bucket=warp-stat3 --duration 30s
```

### Debug

```sh
journalctl -u halmidi.service --since "5 minutes ago" -f
```

### go build
To check whether package is using cgo
```sh
go list -f '{{if .CgoFiles}}{{.ImportPath}} uses cgo{{end}}' all
go list -f '{{.CgoFiles}}' github.com/hashicorp/raft
```

### Notes:
- After installing .deb file logs are adding in /tmp due to permission error
- Changed permission to 777 for base path


### Links
- https://blog.min.io/deprecation-of-the-minio-gateway
- https://hypermode.com/blog/badger
