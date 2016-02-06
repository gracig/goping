package goping

import (
	"testing"
	"time"
)

func TestControllerInjection(t *testing.T) {

	//Test Ctl() for nil
	c := Ctl()
	if c == nil {
		t.Error("Expected Ctl() function to return non nil pointer values")
	}

	//Test if newController() return different pointers
	ca, cb := newController(), newController()
	if ca == cb {
		t.Error("Expected newController() function to return different pointer values per call.")
	}

	//Test the injection of a controller on SetCtl
	SetCtl(ca)
	cc := Ctl()
	if ca != cc {
		t.Error("Expected Ctl() function to return same pointer value of ca after execution of SetCtl(ca)")
	}

}

func TestControllerOSPingerRegistration(t *testing.T) {
	type ospinger struct {
		OSPinger
	}
	type row struct {
		ver  uint8
		prot uint8
		goos string
		osp  *ospinger
		err  error
	}
	table := []*row{
		{IPV4, IPProtoICMP, "linux", new(ospinger), nil},
		{IPV4, IPProtoUDP, "linux", new(ospinger), nil},
		{IPV4, IPProtoTCP, "linux", new(ospinger), nil},
		{IPV6, IPProtoICMP, "linux", new(ospinger), nil},
		{IPV6, IPProtoUDP, "linux", new(ospinger), nil},
		{IPV6, IPProtoTCP, "linux", new(ospinger), nil},
		{IPV4, IPProtoICMP, "darwin", new(ospinger), nil},
		{IPV4, IPProtoUDP, "darwin", new(ospinger), nil},
		{IPV4, IPProtoTCP, "darwin", new(ospinger), nil},
		{IPV6, IPProtoICMP, "darwin", new(ospinger), nil},
		{IPV6, IPProtoUDP, "darwin", new(ospinger), nil},
		{IPV6, IPProtoTCP, "darwin", new(ospinger), nil},
		{IPV4, IPProtoICMP, "windows", new(ospinger), nil},
		{IPV4, IPProtoUDP, "windows", new(ospinger), nil},
		{IPV4, IPProtoTCP, "windows", new(ospinger), nil},
		{IPV6, IPProtoICMP, "windows", new(ospinger), nil},
		{IPV6, IPProtoUDP, "windows", new(ospinger), nil},
		{IPV6, IPProtoTCP, "windows", new(ospinger), nil},
		{IPV4, IPProtoICMP, "freebsd", new(ospinger), nil},
		{IPV4, IPProtoUDP, "freebsd", new(ospinger), nil},
		{IPV4, IPProtoTCP, "freebsd", new(ospinger), nil},
		{IPV6, IPProtoICMP, "freebsd", new(ospinger), nil},
		{IPV6, IPProtoUDP, "freebsd", new(ospinger), nil},
		{IPV6, IPProtoTCP, "freebsd", new(ospinger), nil},
	}

	//traces the memory used by the function
	defer traceMem("TestOSPingerRegistration", statsNow())

	c := Ctl()
	for _, r := range table {
		//Testing the register
		ospr, err := c.RegisterOSPinger(r.ver, r.prot, r.goos, r.osp)
		if r.err != nil {
			if err == nil {
				t.Error("Expected a non nil error, got nil. values:", r)
			}
		} else {
			if err != nil {
				t.Error("Expected a nil error, got non nil. values:", r)
			}
		}
		if ospr != r.osp {
			t.Error("Expected returned pointer value be the same as the parameter. Values", ospr, r.osp)
		}

		//Testing retrieving ospinger
		ospr, err = c.OSPinger(r.ver, r.prot, r.goos)
		if r.err != nil {
			if err == nil {
				t.Error("Expected error on retrieving.", r)
			}
		}
		if ospr != r.osp {
			t.Error("Expected returned pointer value be the same as the parameter. Values", ospr, r.osp)
		}

		//Testing removing ospinger
		ospr, err = c.UnRegisterOSPinger(r.ver, r.prot, r.goos)
		if r.err != nil {
			if err == nil {
				t.Error("Expected error on retrieving.", r)
			}
		}
		if ospr != r.osp {
			t.Error("Expected returned pointer value be the same as the parameter. Values", ospr, r.osp)
		}

		//Testing retrieving a non existent ospinger
		ospr, err = c.OSPinger(r.ver, r.prot, r.goos)
		if err == nil {
			t.Error("Expected Not found error on retrieving.", err, r)
		}
		if ospr != nil {
			t.Error("Expected Nil value for ospr.", err, r)
		}

	}
}

type fakeospinger struct {
	timeToReturn time.Time
}

func (osp *fakeospinger) Ping(r Requester) time.Time {
	return osp.timeToReturn
}

type testRequest struct {
	failBefore bool
	trequest
}

var testRequestTable = []testRequest{
	{false, trequest{
		target: new(target),
	}},
}

func TestControllerPing(t *testing.T) {
	/*
		Convey("Given a fake OSPinger with the Ping function implemented", t, func() {
			osp := new(fakeospinger)
			Convey("When registering it to the  controller", func() {
				Ctl().RegisterOSPinger(IPV4, IPProtoICMP, runtime.GOOS, osp)
				Convey("Then the controller should have it registered", func() {
					ospr, err := Ctl().OSPinger(IPV4, IPProtoICMP, runtime.GOOS)
					So(err, ShouldBeNil)
					So(ospr, ShouldEqual, osp)
				})
			})
		})

		Convey("Given a list of Requests and expected behaviours", t, func() {
			Convey("When calling the controller Ping function for each one of those", func() {
				Convey("Then we should get the expected behaviours", func() {
				})
			})
		})

		type testResponse struct {
			tresponse
			shouldTimeout bool
			shouldFail    bool
		}
		Convey("Given a list of Responses and expected behaviours", t, func() {
			Convey("When calling the controller Pong function for each one of those", func() {
				Convey("Then we should receive the responses and get the expected behaviours", func() {
				})
			})
		})
	*/
}
