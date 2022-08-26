package static

import (
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/markbates/pkger"
)

func Serve() func(ctx *gin.Context) {
	dir := pkger.Dir("/ui/build")
	return static.Serve("/", embeddedFileSystem{dir})
}

type embeddedFileSystem struct {
	pkger.Dir
}

func (p embeddedFileSystem) Exists(_ string, path string) bool {
	_, err := p.Open(path)
	return err == nil
}
