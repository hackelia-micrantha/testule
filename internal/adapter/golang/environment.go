package golang

import (
	"os"
	"path/filepath"
	"runtime"
)

func isolatedGoEnv() ([]string, func(), error) {
	eph, err := os.MkdirTemp("", "testule-go-")
	if err != nil {
		return nil, nil, err
	}
	paths := []string{
		filepath.Join(eph, "home"),
		filepath.Join(eph, "gopath"),
		filepath.Join(eph, "gocache"),
		filepath.Join(eph, "tmp"),
	}
	for _, path := range paths {
		if err := os.Mkdir(path, 0o700); err != nil {
			_ = os.RemoveAll(eph)
			return nil, nil, err
		}
	}
	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + paths[0],
		"GOPATH=" + paths[1],
		"GOMODCACHE=" + filepath.Join(paths[1], "pkg", "mod"),
		"GOCACHE=" + paths[2],
		"GOTMPDIR=" + paths[3],
		"TMPDIR=" + paths[3],
		"GOTOOLCHAIN=local",
		"GOWORK=off",
		"GOPROXY=off",
		"GOSUMDB=off",
		"CGO_ENABLED=0",
		"GOMAXPROCS=2",
		"TZ=UTC",
		"LANG=C.UTF-8",
	}
	if runtime.GOOS == "windows" {
		for _, name := range []string{"SystemRoot", "ComSpec", "PATHEXT", "WINDIR"} {
			if value := os.Getenv(name); value != "" {
				env = append(env, name+"="+value)
			}
		}
	}
	cleanup := func() { _ = os.RemoveAll(eph) }
	return env, cleanup, nil
}
