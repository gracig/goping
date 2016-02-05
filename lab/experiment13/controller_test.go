package goping

import (
	"runtime"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestControllerInjection(t *testing.T) {
	c := newController()
	Convey("Given a controller created by internal function newController", t, func() {
		Convey("When retrieving a controller sc from Ctl()", func() {
			sc := Ctl()
			Convey("Then c should NOT equals sc", func() {
				So(c, ShouldNotEqual, sc)
			})
		})
		Convey("When invoking SetCtl(c)", func() {
			SetCtl(c)
			Convey("When retrieving a controller sc from Ctl()", func() {
				sc := Ctl()
				Convey("Then c should equals sc", func() {
					So(c, ShouldEqual, sc)
				})
			})
		})
	})
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
	}
	table := []*row{
		{IPV4, IPProtoICMP, "linux", new(ospinger)},
		{IPV4, IPProtoUDP, "linux", new(ospinger)},
		{IPV4, IPProtoTCP, "linux", new(ospinger)},
		{IPV6, IPProtoICMP, "linux", new(ospinger)},
		{IPV6, IPProtoUDP, "linux", new(ospinger)},
		{IPV6, IPProtoTCP, "linux", new(ospinger)},
		{IPV4, IPProtoICMP, "darwin", new(ospinger)},
		{IPV4, IPProtoUDP, "darwin", new(ospinger)},
		{IPV4, IPProtoTCP, "darwin", new(ospinger)},
		{IPV6, IPProtoICMP, "darwin", new(ospinger)},
		{IPV6, IPProtoUDP, "darwin", new(ospinger)},
		{IPV6, IPProtoTCP, "darwin", new(ospinger)},
		{IPV4, IPProtoICMP, "windows", new(ospinger)},
		{IPV4, IPProtoUDP, "windows", new(ospinger)},
		{IPV4, IPProtoTCP, "windows", new(ospinger)},
		{IPV6, IPProtoICMP, "windows", new(ospinger)},
		{IPV6, IPProtoUDP, "windows", new(ospinger)},
		{IPV6, IPProtoTCP, "windows", new(ospinger)},
		{IPV4, IPProtoICMP, "freebsd", new(ospinger)},
		{IPV4, IPProtoUDP, "freebsd", new(ospinger)},
		{IPV4, IPProtoTCP, "freebsd", new(ospinger)},
		{IPV6, IPProtoICMP, "freebsd", new(ospinger)},
		{IPV6, IPProtoUDP, "freebsd", new(ospinger)},
		{IPV6, IPProtoTCP, "freebsd", new(ospinger)},
	}

	defer traceMem("TestOSPingerRegistration", statsNow())
	Convey("Given a controller c retrieved by singleton function Ctl()", t, func() {
		c := Ctl()
		Convey("When a controller sc is retrieved by sigleton function Ctl()", func() {
			sc := Ctl()
			Convey("Then sc should be equal c", func() {
				So(sc, ShouldEqual, c)
			})
		})
		Convey("Given a list of  ospinger to register", func() {
			Convey("When calling c.ResgisterOSPinger() for each of them", func() {
				Convey("Then returned error should be nil and returned ospinger should be equals ospinger", func() {
					for _, r := range table {
						ospr, err := c.RegisterOSPinger(r.ver, r.prot, r.goos, r.osp)
						So(err, ShouldBeNil)
						So(ospr, ShouldEqual, r.osp)
					}
				})
			})
		})
		Convey("Given a list of ospinger already registered", func() {
			Convey("When calling c.OSPinger() for each of them", func() {
				Convey("Then returned error should be nil and returned ospinger should be equals ospinger", func() {
					for _, r := range table {
						ospr, err := c.OSPinger(r.ver, r.prot, r.goos)
						So(err, ShouldBeNil)
						So(ospr, ShouldEqual, r.osp)
					}
				})
			})
		})
		Convey("Given a list of ospinger already registered and retrieved", func() {
			Convey("When calling c.UnRegisterOSPinger()", func() {
				Convey("Then returned error should be nil and returned ospinger should be equals ospinger", func() {
					for _, r := range table {
						ospr, err := c.UnRegisterOSPinger(r.ver, r.prot, r.goos)
						So(err, ShouldBeNil)
						So(ospr, ShouldEqual, r.osp)
					}
				})
			})
		})
		Convey("Given a list of ospinger already deleted", func() {
			Convey("When calling c.OSPinger()", func() {
				Convey("Then returned error should NOT be nil and returned ospinger should be nil", func() {
					for _, r := range table {
						ospr, err := c.OSPinger(r.ver, r.prot, r.goos)
						So(err, ShouldNotBeNil)
						So(ospr, ShouldBeNil)
					}
				})
			})
		})
	})
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
}
