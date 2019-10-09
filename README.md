# README

微信网页授权微服务

部署完可以测试以下URL

curl 或者 微信开发者工具里

需要去微信公众号后台设置授权域名（公众号设置 -> 功能设置 -> 网页授权域名）

假设授权域名成功设置为了 wx.example.com

Verify Domain   
https://wx.example.com/MP_verify_YxLL1ZR1olEyFcp0.txt

签名   
https://wx.example.com/jssdk_signature


开始网页授权   
https://wx.example.com/auth

\rsync -avP . aliyun.zdp:~/xsjd

docker-compose up
