/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package runc

import (
	"io"
	"os"
	"os/exec"
)

type IO interface {
	io.Closer
	Stdin() io.WriteCloser
	Stdout() io.ReadCloser
	Stderr() io.ReadCloser
	Set(*exec.Cmd)
}

type StartCloser interface {
	CloseAfterStart() error
}

func newPipe() (*pipe, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	return &pipe{
		r: r,
		w: w,
	}, nil
}

type pipe struct {
	r *os.File
	w *os.File
}

func (p *pipe) Close() error {
	err := p.r.Close()
	if werr := p.w.Close(); err == nil {
		err = werr
	}
	return err
}

type pipeIO struct {
	in  *pipe
	out *pipe
	err *pipe
}

func (i *pipeIO) Stdin() io.WriteCloser {
	return i.in.w
}

func (i *pipeIO) Stdout() io.ReadCloser {
	return i.out.r
}

func (i *pipeIO) Stderr() io.ReadCloser {
	return i.err.r
}

func (i *pipeIO) Close() error {
	var err error
	for _, v := range []*pipe{
		i.in,
		i.out,
		i.err,
	} {
		if cerr := v.Close(); err == nil {
			err = cerr
		}
	}
	return err
}

func (i *pipeIO) CloseAfterStart() error {
	for _, f := range []*os.File{
		i.out.w,
		i.err.w,
	} {
		f.Close()
	}
	return nil
}

// Set sets the io to the exec.Cmd
func (i *pipeIO) Set(cmd *exec.Cmd) {
	cmd.Stdin = i.in.r
	cmd.Stdout = i.out.w
	cmd.Stderr = i.err.w
}

func NewSTDIO() (IO, error) {
	return &stdio{}, nil
}

type stdio struct {
}

func (s *stdio) Close() error {
	return nil
}

func (s *stdio) Set(cmd *exec.Cmd) {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
}

func (s *stdio) Stdin() io.WriteCloser {
	return os.Stdin
}

func (s *stdio) Stdout() io.ReadCloser {
	return os.Stdout
}

func (s *stdio) Stderr() io.ReadCloser {
	return os.Stderr
}

// NewNullIO returns IO setup for /dev/null use with runc
func NewNullIO() (IO, error) {
	f, err := os.Open(os.DevNull)
	if err != nil {
		return nil, err
	}
	return &nullIO{
		devNull: f,
	}, nil
}

type nullIO struct {
	devNull *os.File
}

func (n *nullIO) Close() error {
	// this should be closed after start but if not
	// make sure we close the file but don't return the error
	n.devNull.Close()
	return nil
}

func (n *nullIO) Stdin() io.WriteCloser {
	return nil
}

func (n *nullIO) Stdout() io.ReadCloser {
	return nil
}

func (n *nullIO) Stderr() io.ReadCloser {
	return nil
}

func (n *nullIO) Set(c *exec.Cmd) {
	// don't set STDIN here
	c.Stdout = n.devNull
	c.Stderr = n.devNull
}

func (n *nullIO) CloseAfterStart() error {
	return n.devNull.Close()
}
