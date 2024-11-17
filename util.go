package pdfire

import (
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func changeOwnerPassword(r io.ReadSeeker, w io.Writer, pwOld, pwNew string, conf *pdfcpu.Configuration) error {
	conf.Cmd = pdfcpu.CHANGEOPW
	conf.OwnerPW = pwOld
	conf.OwnerPWNew = &pwNew
	return api.Optimize(r, w, conf)
}

func changeUserPassword(r io.ReadSeeker, w io.Writer, pwOld, pwNew string, conf *pdfcpu.Configuration) error {
	conf.Cmd = pdfcpu.CHANGEUPW
	conf.UserPW = pwOld
	conf.UserPWNew = &pwNew
	return api.Optimize(r, w, conf)
}
