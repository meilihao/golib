# virt
推荐手动构建xml:
- 优点

    1. 当所有可引导设备都没有boot order时, libvirt自动添加`<os><boot dev='...'/></os>`, 与之后添加的带boot order属性的设备冲突
- 缺点:

    1. 没有virtinst项目帮忙检查参数

## deps
```bash
apt install virtinst libosinfo-bin # virtinst is virt-install, libosinfo-bin is osinfo-query
```

## FAQ
即使/etc/libvirt/qemu/xxx.xml对应的vm没有启动, 手动删除xxx.xml后创建相同配置的vm还是会报错,提示`...  已被其他客户机 ['xxx'] 使用`, 因此推测libvirtd有cache, 需要先调用`Undefine()`