# virt

## deps
```bash
apt install virtinst libosinfo-bin # virtinst is virt-install, libosinfo-bin is osinfo-query
```

## FAQ
即使/etc/libvirt/qemu/xxx.xml对应的vm没有启动, 手动删除xxx.xml后创建相同配置的vm还是会报错,提示`...  已被其他客户机 ['xxx'] 使用`, 因此推测libvirtd有cache, 需要先调用`Undefine()`