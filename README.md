# vanilla-mihomo
windows mihomo裸核运行配置与一键执行脚本

脚本为系统代理模式，tun自行修改

## 系统代理模式
- 下载[mihomo裸核](https://github.com/MetaCubeX/mihomo/releases)
- 解压到任一文件夹，并将命名mihomo.exe
- 自定义配置[config.yaml](https://github.com/HenryChiao/MIHOMO_YAMLS)文件,配置文件中当前端口设置为7890，若更改，则[toggle_mihomo.vbs](https://github.com/hgoev/vanilla-mihomo/blob/main/toggle_mihomo.vbs)中36行端口也随之更改
- 配置完毕config文件后，双击toggle_mihomo.vbs即可
- http://127.0.0.1:9090/ui 查看web页面

## tun模式
- 下载[mihomo裸核](https://github.com/MetaCubeX/mihomo/releases)
- 解压到任一文件夹，并将命名mihomo.exe
- 自定义配置[config.yaml](https://github.com/HenryChiao/MIHOMO_YAMLS)文件,下载支持tun配置文件
- 配置完毕config文件后，以管理员身份运行.\mihomo.exe -d "." -f ".\config.yaml"
- http://127.0.0.1:9090/ui 查看web页面
