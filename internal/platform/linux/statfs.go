//go:build linux

package linux

import "syscall"

func statfs(path string) (total, used uint64, err error) {
	var st syscall.Statfs_t
	if err = syscall.Statfs(path, &st); err != nil {
		return
	}
	bs := uint64(st.Bsize)
	total = uint64(st.Blocks) * bs
	used = total - uint64(st.Bfree)*bs
	return
}
