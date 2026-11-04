### 表迁移： go run . -db

### prometheus.exe // 启动普罗米修斯
### grafana.exe server // 启动grafana

### 上传本项目到生产
1. .\scripts\build.ps1
2. 编译完顺手验证嵌入了什么：go version -m .\go-star-linux.new | Select-String "vcs"
    预期看到：build  vcs.revision=...哈希值
             build  vcs.modified=false
    vcs.modified=false 说明工作区干净——这也顺便证明 scripts/ 和画像确实都提交进去了。
3. 上传（🖥️ 本地）
    scp go-star-linux.new root@真实ip地址:/home/admin/go-star/
4. 部署（🐧 服务器）
    cd /home/admin/go-star && ./deploy.sh
    若线上的md5与本地的一样，输入yes
5. 最后的健康检查是关键：如果新进程起不来，脚本会自动回滚到刚才的备份，你不用手动救。
