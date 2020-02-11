package winlib_test

import (
	"fmt"
	"ivpn/daemon/service/firewall/winlib"
	"syscall"
	"testing"
)

func TestFuncsCall(t *testing.T) {
	DoTest()
}

func DoTest() {
	defer func() {
		if r := recover(); r != nil {

		}
	}()

	sess, errS1 := winlib.CreateWfpSessionObject(true)
	fmt.Println(sess, errS1)

	engine, errEn1 := winlib.WfpEngineOpen(sess)
	fmt.Println(engine, errEn1)

	guid := syscall.GUID{
		Data1: 0xfed0afd4,
		Data2: 0x98d4,
		Data3: 0x4233,
		Data4: [8]byte{0xa4, 0xf3, 0x8b, 0x7c, 0x02, 0x44, 0x50, 0x01},
	}

	isInstalled, flags, err := winlib.WfpGetProviderFlags(engine, guid)
	fmt.Println(isInstalled, flags, err)

	sublayer, serr := winlib.FWPMSUBLAYER0Create(guid)
	fmt.Println(sublayer, serr)

	isInstalled, err = winlib.WfpSubLayerIsInstalled(engine, guid)
	fmt.Println(isInstalled, err)

	winlib.FWPMSUBLAYER0SetDisplayData(sublayer, "sbName", "sbDescription")
	fmt.Println(err)

	err = winlib.WfpSubLayerAdd(engine, sublayer)
	fmt.Println(err)

	isInstalled, err = winlib.WfpSubLayerIsInstalled(engine, guid)
	fmt.Println(isInstalled, err)

	winlib.WfpSubLayerDelete(engine, guid)
	fmt.Println(err)

	provider, perr := winlib.FWPMPROVIDER0Create(guid)
	fmt.Println(provider, perr)

	err = winlib.FWPMPROVIDER0SetFlags(provider, 1)
	fmt.Println(err)

	err = winlib.FWPMPROVIDER0SetDisplayData(provider, "theName", "theDESC")
	fmt.Println(err)

	err = winlib.WfpProviderAdd(engine, provider)
	fmt.Println(err)

	err = winlib.WfpProviderDelete(engine, guid)
	fmt.Println(err)

	isInstalled, flags, err = winlib.WfpGetProviderFlags(engine, guid)
	fmt.Println(isInstalled, flags, err)

	err = winlib.WfpTransactionCommit(engine)
	fmt.Println(err)

	err = winlib.WfpTransactionAbort(engine)
	fmt.Println(err)

	err = winlib.WfpTransactionBegin(engine)
	fmt.Println(err)
	err = winlib.WfpTransactionCommit(engine)
	fmt.Println(err)

	err = winlib.WfpTransactionBegin(engine)
	fmt.Println(err)
	err = winlib.WfpTransactionAbort(engine)
	fmt.Println(err)

	errEC := winlib.WfpEngineClose(engine)
	fmt.Println(errEC)

	winlib.DeleteWfpSessionObject(sess)
	return
}
