## WeeklyReport ##
##### golang构建的公司部门周报报送的websocket项目 #####



#### 安装 ####
```shell
go get github.com/W-Jie/weeklyreport
cd $GOPATH/src/github.com/W-Jie/weeklyreport
go bulid
```

#### Usage ####
```shell
# 复制并修改配置文件
cp config.yaml.example config.yaml

# 运行并重定向日志
./weeklyreport >> weeklyreport.log
```

#### Todo####
- 增加日志输出
- 增加metrics