package hosts

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/sagernet/sing-box/log"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/service/filemanager"

	"github.com/miekg/dns"
)

const cacheMaxAge = 5 * time.Second

type File struct {
	ctx     context.Context
	logger  log.ContextLogger
	path    string
	access  sync.Mutex
	byName  map[string][]netip.Addr
	expire  time.Time
	modTime time.Time
	size    int64
}

func NewFile(ctx context.Context, path string) *File {
	return NewFileWithLogger(ctx, nil, path)
}

func NewFileWithLogger(ctx context.Context, logger log.ContextLogger, path string) *File {
	return &File{
		ctx:    ctx,
		logger: logger,
		path:   path,
	}
}

func NewDefault() (*File, error) {
	return NewDefaultWithLogger(nil)
}

func NewDefaultWithLogger(logger log.ContextLogger) (*File, error) {
	defaultPathResolved, err := defaultPath()
	if err != nil {
		return nil, E.Cause(err, "resolve default hosts path")
	}
	return NewFileWithLogger(context.Background(), logger, defaultPathResolved), nil
}

func (f *File) Lookup(name string) []netip.Addr {
	f.access.Lock()
	defer f.access.Unlock()
	f.update()
	addrs := f.byName[dns.CanonicalName(name)]
	if len(addrs) > 0 && f.logger != nil {
		f.logger.DebugContext(f.ctx, "hosts lookup hit: ", name, " -> ", addrs)
	}
	return addrs
}

func (f *File) update() {
	now := time.Now()
	if now.Before(f.expire) && len(f.byName) > 0 {
		return
	}
	stat, err := filemanager.Stat(f.ctx, f.path)
	if err != nil {
		if f.logger != nil {
			f.logger.WarnContext(f.ctx, "stat hosts file failed: ", f.path, ": ", err)
		}
		return
	}
	if f.modTime.Equal(stat.ModTime()) && f.size == stat.Size() {
		f.expire = now.Add(cacheMaxAge)
		return
	}
	byName := make(map[string][]netip.Addr)
	file, err := filemanager.Open(f.ctx, f.path)
	if err != nil {
		if f.logger != nil {
			f.logger.WarnContext(f.ctx, "open hosts file failed: ", f.path, ": ", err)
		}
		return
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	var (
		prefix   []byte
		line     []byte
		isPrefix bool
	)
	for {
		line, isPrefix, err = reader.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			if f.logger != nil {
				f.logger.WarnContext(f.ctx, "read hosts file error: ", f.path, ": ", err)
			}
			return
		}
		if isPrefix {
			prefix = append(prefix, line...)
			continue
		} else if len(prefix) > 0 {
			line = append(prefix, line...)
			prefix = nil
		}
		commentIndex := strings.IndexRune(string(line), '#')
		if commentIndex != -1 {
			line = line[:commentIndex]
		}
		fields := strings.Fields(string(line))
		if len(fields) < 2 {
			continue
		}
		var addr netip.Addr
		addr, err = netip.ParseAddr(fields[0])
		if err != nil {
			if f.logger != nil {
				f.logger.DebugContext(f.ctx, "skip invalid IP in hosts file ", f.path, ": ", fields[0])
			}
			continue
		}
		for index := 1; index < len(fields); index++ {
			canonicalName := dns.CanonicalName(fields[index])
			byName[canonicalName] = append(byName[canonicalName], addr)
		}
	}
	f.expire = now.Add(cacheMaxAge)
	f.modTime = stat.ModTime()
	f.size = stat.Size()
	f.byName = byName
	if f.logger != nil {
		f.logger.InfoContext(f.ctx, "loaded ", len(byName), " hosts domains from ", f.path)
		for domain, addrs := range byName {
			f.logger.InfoContext(f.ctx, "hosts entry: ", domain, " -> ", addrs)
		}
	}
}
