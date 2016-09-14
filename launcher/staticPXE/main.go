package main

import (
	"net"
	"io/ioutil"
	"os/exec"
	"bytes"
	"strconv"
	"hash/crc32"
	"encoding/binary"
	"encoding/json"
	"log"
	"flag"
	"os"
	"time"
)

func ipAdd(start net.IP, add int) (result net.IP) {
	result = make(net.IP, 4);
	binary.BigEndian.PutUint32(result, binary.BigEndian.Uint32(start.To4())+uint32(add));
	return;
}

//port from MAC
func portHA(mac net.HardwareAddr /*, _ ...error /*black hole for errors */) string {
	var macCRC32 = crc32.Checksum(mac, crc32.MakeTable(crc32.Koopman));

	return strconv.FormatUint(uint64(((macCRC32>>16)^(macCRC32&0xFFFF))|0x8000), 10);   // port#: 0x8000..0xFFFF
}

//port from MAC
func portS(mac string) string{
	if macHA,err := net.ParseMAC(mac); err != nil {
		log.Fatal("Error can't parse MAC:", err);
	} else {
		return portHA(macHA);
	}
	return "";
}

func instanceNumber(inter net.Interface, MACs []string) (n int) {
	for n=0; n<len(MACs); n++ {
		if mac,err := net.ParseMAC(MACs[n]); err != nil {
			log.Fatal("Error can't parse MAC:", err);
		} else {
			if bytes.Equal(inter.HardwareAddr, mac) { return; }
		}
	}

	// n == len(MACs)
	log.Fatal("Turn off PC (can't find this computer MAC in config file).");
	return;
}

func setIP(inter net.Interface, ip, mask net.IP){
	var netshCmd = exec.Command("netsh", "interface", "ipv4", "set", "address",
		`name="`+/*inter.Name*/"Ethernet"+`"`,
		"source=static",
		"address="+ip.String(),
		"mask="+mask.String());
	netshCmd.Stderr = os.Stderr;
	netshCmd.Stdout = os.Stdout;

	if err := netshCmd.Run(); err != nil {
		log.Fatal("Error set IP ", err);
	}
}

func setARP(inter net.Interface, ip net.IP, MAC string){
	var netshCmd = exec.Command("netsh", "interface", "ipv4", "set", "neighbors",
		`interface="`+/*inter.Name*/"Ethernet"+`"`,
		"address="+ip.String(),
		`neighbor="`+ MAC +`"`);
	netshCmd.Stderr = os.Stderr;
	netshCmd.Stdout = os.Stdout;

	if err := netshCmd.Run(); err != nil {
		log.Fatal("Error set ARP ", err);
	}
}


func main() {
	var conf = struct{
		file     *string;
		StartIP   net.IP  `json:"startIP"`;
		Mask      net.IP  `json:"mask"`;
		MAC     []string  `json:"mac"`;
	}{};

	{
		var configPath =  flag.String("conf", "example_config.json", "path to config file");
		conf.file      =  flag.String("file", "", "path to transfer file");

		flag.Parse();

		if *conf.file == "" {
			log.Fatalln("Please set path to transfer file.");
		}

		var configJSON, err = ioutil.ReadFile(*configPath);
		if err != nil {
			log.Fatalln("Error reading config file:", err);
		}

		if err = json.Unmarshal(configJSON, &conf); err != nil {
			log.Fatalln("Error parsing config file:", err);
		}
	}

	var inter net.Interface;
	{
		var interfaces, err = net.Interfaces();
		if err != nil {
			log.Fatal("Error can't get Interfaces: ", err);
		}
		inter = interfaces[0];
	}

	var instNum = instanceNumber(inter, conf.MAC);

	{
		setIP(inter, ipAdd(conf.StartIP, instNum), conf.Mask);

		if instNum != 0 {
			setARP(inter, ipAdd(conf.StartIP, instNum-1), conf.MAC[instNum-1]);  //set seed IP-MAC
		}
		if instNum != len(conf.MAC)-1 {
			setARP(inter, ipAdd(conf.StartIP, instNum+1), conf.MAC[instNum+1]);  //set leech IP-MAC
		}
	}

	var cmdRingSync *exec.Cmd;
	switch(instNum){
	case 0: //seed
		cmdRingSync = exec.Command("RingSync.exe", "-mode=seed",
			"-port="  + portHA(inter.HardwareAddr),
			"-leech=" + ipAdd(conf.StartIP, instNum+1).String(),
			"-if="    + *conf.file);

	case len(conf.MAC)-1: //leech
		cmdRingSync = exec.Command("RingSync.exe", "-mode=leech",
			"-seed=" + ipAdd(conf.StartIP, instNum-1).String() + ":" + portS(conf.MAC[instNum-1]),
			"-of="   + *conf.file);

	default: //peer
		cmdRingSync = exec.Command("RingSync.exe", "-mode=peer",
			"-port="  + portHA(inter.HardwareAddr),
			"-leech=" + ipAdd(conf.StartIP, instNum+1).String(),
			"-seed="  + ipAdd(conf.StartIP, instNum-1).String() + ":" + portS(conf.MAC[instNum-1]),
			"-of="    + *conf.file);
	}
	cmdRingSync.Stderr = os.Stderr;
	cmdRingSync.Stdout = os.Stdout;

	//TODO: use more correct solution
	time.Sleep(10*time.Second);  //wait to system apply network changes (prevent bug: dial to unreacheble network)

	if err := cmdRingSync.Start(); err != nil {
		log.Fatal("Error start RingSync ", err);
	}

}
