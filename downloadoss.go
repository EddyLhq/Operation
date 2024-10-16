//go1.20.1
//export OSS_ACCESS_KEY_ID=
//export OSS_ACCESS_KEY_SECRET=
package main

import (
    "log"
    "time"
    "fmt"
    "strings"
    "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

func main() {
    // 从环境变量中获取访问凭证。运行本代码示例之前，请确保已设置环境变量OSS_ACCESS_KEY_ID和OSS_ACCESS_KEY_SECRET。
    provider, err := oss.NewEnvironmentVariableCredentialsProvider()
    if err != nil {
        log.Fatalf("Failed to create credentials provider: %v", err)
    }

    // 创建OSSClient实例。
    // yourEndpoint填写Bucket对应的Endpoint，以华东1（杭州）为例，填写为https://oss-cn-hangzhou.aliyuncs.com。其它Region请按实际情况填写。
    endpoint := "https://oss-ap-southeast-5.aliyuncs.com" // 请替换为实际的Endpoint
    client, err := oss.New(endpoint, "", "", oss.SetCredentialsProvider(&provider))
    if err != nil {
        log.Fatalf("Failed to create OSS client: %v", err)
    }

    // 填写存储空间名称。
    bucketName := "idn-web" // 请替换为实际的Bucket名称
    bucket, err := client.Bucket(bucketName)
    if err != nil {
        log.Fatalf("Failed to get bucket: %v", err)
    }

    //本地存放下载文件路径
    localFilePath := "/data/obs/"
    // 分页列举包含指定前缀的文件。默认情况下，一次返回100条记录。
    continueToken := ""
    prefix := "portrait"
    for {
        lsRes, err := bucket.ListObjectsV2(oss.Prefix(prefix), oss.MaxKeys(50),  oss.ContinuationToken(continueToken))
        if err != nil {
            log.Fatalf("Failed to list objects: %v", err)
        }

        // 打印列举结果。
        for _, object := range lsRes.Objects {
            if strings.HasSuffix(object.Key, "/") {
               log.Printf("Ignoring file: %s\n", object.Key)
               continue
            }
            log.Printf("Object Key: %s, Type: %s, Size: %d, ETag: %s, LastModified: %s, StorageClass: %s\n",
                object.Key, object.Type, object.Size, object.ETag, object.LastModified.Format(time.RFC3339), object.StorageClass)
            LocalFileName := localFilePath + object.Key
            err = bucket.GetObjectToFile(object.Key, LocalFileName)
            if err != nil {
                fmt.Println("错误:",err)
            }
        }

        // 如果结果被截断，则更新继续标记并继续循环
        if lsRes.IsTruncated {
            continueToken = lsRes.NextContinuationToken
        } else {
            break
        }
    }

    log.Println("All objects have been listed.")
}
