插件私有依赖说明
================

> monstcache插件基于golang plugin机制实现，该机制要求主体和插件编译依赖组件必须一致且不兼容vendor和go mod混合，
> 故此在此处单独做私有化依赖管理。

- 此vendor下依赖为monstcache插件私有依赖，除github.com/rwynn下组件外都为开源标准组件未做私有改动;
- monstcache版本为私有化版本（修复了bug和增加插件特性，社区官方暂无人维护），编译出包时会在此vendor内进行monstcache和插件的编译;
- 系统部分package引入了blog，其中blog参数和monstcache冲突造成启动异常，在私有版本中修改了monstcache命令行参数v;
