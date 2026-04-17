package main

import (
	"fmt"
	"path/filepath"
    "os"
	obs "github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
)
//上传文件到obs
func putFileOBS(filePath string, obsDirPath string ) error {
    //推荐通过环境变量获取AKSK，这里也可以使用其他外部引入方式传入，如果使用硬编码可能会存在泄露风险。
    //您可以登录访问管理控制台获取访问密钥AK/SK，获取方式请参见https://support.huaweicloud.com/intl/zh-cn/usermanual-ca/ca_01_0003.html。
    obsEndpoint := ""
    obsAccessKeySecret := ""
    obsAccessKeyID := ""
    bucketName := ""

    //ak := os.Getenv("AccessKeyID")
    //sk := os.Getenv("SecretAccessKey")
    // endpoint填写Bucket对应的Endpoint, 这里以中国-香港为例，其他地区请按实际情况填写。
    //endPoint := "https://obs.ap-southeast-1.myhuaweicloud.com"
    // 创建obsClient实例
    // 如果使用临时AKSK和SecurityToken访问OBS，需要在创建实例时通过obs.WithSecurityToken方法指定securityToken值。
    obsClient, err := obs.New(obsAccessKeyID, obsAccessKeySecret, obsEndpoint)
    if err != nil {
        fmt.Printf("Create obsClient error, errMsg: %s", err.Error())
        return err
    }
    // 指定存储桶名称
    input := &obs.PutFileInput{}
    input.Bucket = bucketName
    // 指定上传对象，此处以 example/objectname 为例。
    input.Key = obsDirPath
    // 指定本地文件，此处以localfile为例
    input.SourceFile = filePath
    // 文件上传
    output, err := obsClient.PutFile(input)
    if err == nil {
        fmt.Printf("Put file(%s) under the bucket(%s) successful!\n", input.Key, input.Bucket)
        fmt.Printf("StorageClass:%s, ETag:%s\n",
            output.StorageClass, output.ETag)
    }else {
        return err
    }
    return nil
}

func main() {
    localPath := "/data/obs/portrait"
    err := filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        if !info.IsDir() {
            // 获取完整文件路径名称
            obsRelativePath, err := filepath.Rel("/data/obs/", path)
            fmt.Printf("待上传的文件名: %s\n", obsRelativePath)
            if err != nil {
                return err
            }
            err = putFileOBS(path, obsRelativePath)
            if err != nil {
                return err
            }else {
                fmt.Printf("上传的文件列表: %s\n", path)
            }					    
        }				
        return nil				
    })
    if err != nil {
        fmt.Printf("上传出错，错误原因：%s\n",err)
        return
    }

}
