/* {{{ Copyright (c) Paul R. Tagliamonte <paultag@debian.org>, 2015
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE. }}} */

package archive

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"hash"

	"pault.ag/go/debian/control"
	"pault.ag/go/debian/deb"
	"pault.ag/go/debian/dependency"
	"pault.ag/go/debian/version"
)

// Package {{{

// Binary .deb Package entry, as it exists in the Packages file, which
// contains the .deb Control information, as well as information on
// where the file lives, the file size, and some hashes.
type Package struct {
	control.Paragraph

	Package       string `required:"true"`
	Source        string
	Version       version.Version `required:"true"`
	Section       string
	Priority      string
	Architecture  dependency.Arch `required:"true"`
	Essential     string
	InstalledSize int    `control:"Installed-Size"`
	Maintainer    string `required:"true"`
	Description   string `required:"true"`
	Homepage      string

	Filename       string `required:"true"`
	Size           int    `required:"true"`
	MD5sum         string
	SHA1           string
	SHA256         string
	SHA512         string
	DescriptionMD5 string `control:"Description-md5"`
}

// Depends helpers {{{

// Get a dependency relation on the fly, given a key. This is just to reduce
// redundency on the methods below.
func getDepends(paragraph control.Paragraph, key string) (*dependency.Dependency, error) {
	if val, ok := paragraph.Values[key]; ok {
		return dependency.Parse(val)
	}
	return &dependency.Dependency{}, nil
}

// Parse the Depends Dependency relation on this package.
func (p Package) Depends() (*dependency.Dependency, error) {
	return getDepends(p.Paragraph, "Depends")
}

// Parse the Depends Suggests relation on this package.
func (p Package) Suggests() (*dependency.Dependency, error) {
	return getDepends(p.Paragraph, "Suggest")
}

// Parse the Depends Built-Using relation on this package.
func (p Package) BuiltUsing() (*dependency.Dependency, error) {
	return getDepends(p.Paragraph, "Built-Using")
}

// Parse the Depends Breaks relation on this package.
func (p Package) Breaks() (*dependency.Dependency, error) {
	return getDepends(p.Paragraph, "Breaks")
}

// Parse the Depends Replaces relation on this package.
func (p Package) Replaces() (*dependency.Dependency, error) {
	return getDepends(p.Paragraph, "Replaces")
}

// Parse the Depends Pre-Depends relation on this package.
func (p Package) PreDepends() (*dependency.Dependency, error) {
	return getDepends(p.Paragraph, "Pre-Depends")
}

// }}}

// PackageFromDeb {{{

// Create a Package entry from a deb.Deb file. This will copy the binary
// .deb Control file into the Package entry, and set information as to
// the location of the file, the size of the file, and hash the file.
func PackageFromDeb(debFile deb.Deb) (*Package, error) {
	pkg := Package{}

	paragraph := debFile.Control.Paragraph
	paragraph.Set("Filename", debFile.Path)
	/* Now, let's do some magic */

	fd, err := os.Open(debFile.Path)
	if err != nil {
		return nil, err
	}
	stat, err := fd.Stat()
	if err != nil {
		return nil, err
	}

	paragraph.Set("Size", strconv.Itoa(int(stat.Size())))
	/* Right, now, in addition, we ought to hash the crap out of the file */

	md5sum := md5.New()
	sha1 := sha1.New()
	sha256 := sha256.New()

	writer := io.MultiWriter(md5sum, sha256, sha1)

	if _, err := io.Copy(writer, fd); err != nil {
		return nil, err
	}

	for key, hasher := range map[string]hash.Hash{
		"MD5sum": md5sum,
		"SHA1":   sha1,
		"SHA256": sha256,
	} {
		paragraph.Set(key, fmt.Sprintf("%x", hasher.Sum(nil)))
	}

	return &pkg, control.UnpackFromParagraph(debFile.Control.Paragraph, &pkg)
}

// }}}

// }}}

// Packages {{{

// Iterator to access the entries contained in the Packages entry in an
// apt repo. This contians information about the binary Debian packages.
type Packages struct {
	decoder *control.Decoder
}

// Next {{{

// Get the next Package entry in the Packages list. This will return an
// io.EOF at the last entry.
func (p *Packages) Next() (*Package, error) {
	next := Package{}
	return &next, p.decoder.Decode(&next)
}

// }}}

// LoadPackagesFile {{{

// Given a path, create a Packages iterator. Note that the Packages
// file is not OpenPGP signed, so one will need to verify the integrety
// of this file from the InRelease file before trusting any output.
func LoadPackagesFile(path string) (*Packages, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return LoadPackages(fd)
}

// }}}

// LoadPackages {{{

// Given an io.Reader, create a Packages iterator. Note that the Packages
// file is not OpenPGP signed, so one will need to verify the integrety
// of this file from the InRelease file before trusting any output.
func LoadPackages(in io.Reader) (*Packages, error) {
	decoder, err := control.NewDecoder(in, nil)
	if err != nil {
		return nil, err
	}
	return &Packages{decoder: decoder}, nil
}

// }}}

// }}}

// vim: foldmethod=marker
