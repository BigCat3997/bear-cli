package models

type AWSCredential struct {
	AccessKeyId     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	Region          string `json:"region"`
}
