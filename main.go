package main

// Project Iris - Completely decentralized cloud messaging http://iris.karalabe.com/book/run_forrest_run
// https://github.com/elgs/filesync

// Недостатки TCP и новые протоколы транспортного уровня http://habrahabr.ru/post/230863
// Fast Data Transfers http://www.catapultsoft.com
// Pulse - distributed file synchronisation engine https://ind.ie/labs

import (
	"net"
	"io"
	"os"
	"log"
	"fmt"
	"flag"

	"github.com/cheggaaa/pb"
)

func listen(port string) net.Listener {
	l, err := net.Listen("tcp", ":"+port);
	if err != nil {
		log.Fatal("Error listening:", err.Error());
	}
	log.Println("Listening on 0.0.0.0:" + port);

	return l;
}

func accept(l net.Listener, leechAddr string) net.Conn {
	var c net.Conn;
	var err error;

	for {
		c, err = l.Accept();
		if err != nil {
			log.Fatal("Error accepting: ", err.Error());
		}
		// if c.RemoteAddr().(net.TCPAddr).IP.Equal(leechAddr) { break; }
		if host,_,_ := net.SplitHostPort(c.RemoteAddr().String()); host == leechAddr { break; }

		log.Println("Rejected ", c.RemoteAddr());
		c.Close();
	}

	log.Println("Accepted ", c.RemoteAddr());

	return c;
}

func dial(address string) net.Conn {
	c, err := net.Dial("tcp", address);
	if err != nil {
		log.Fatal("Error dialing:", err.Error());
	}
	log.Println("Dialed " + address);

	return c;
}

func transfer(out io.Writer, in io.Reader){
	n, err := io.Copy(out, in);
	if err != nil {
		log.Fatal("Error transfer: ", err.Error());
	}
	log.Println("Transfer ", n, " bytes");
}

// Sync and Close file
func flushFile(file *os.File){
	log.Println("Flushing to disk...");

	file.Sync();
	file.Close();

	log.Println("Flushed!");
}

func seed(transferFileName, listenPort, leechAddr string){
	inputFile, err := os.Open(transferFileName);
	if err != nil { log.Fatal("Error open file: ", err.Error()); }
	defer inputFile.Close();

	var listener = listen(listenPort);
	defer listener.Close();

	var peerConn = accept(listener, leechAddr);
	defer peerConn.Close();
	listener.Close();

	var fileInfo,_ = inputFile.Stat();
	var bar = pb.New64(fileInfo.Size()).SetUnits(pb.U_BYTES);
	bar.ShowSpeed = true;
	bar.Start();

	transfer(io.MultiWriter(peerConn, bar), inputFile);

	bar.FinishPrint("Transfer successfully completed!");
}

func peer(transferFileName, listenPort, seedAddr, leechAddr string){
	outFile, err := os.Create(transferFileName);
	if err != nil { log.Fatal("Error create file: ", err.Error()); }
	defer flushFile(outFile);

	var listener = listen(listenPort);
	defer listener.Close();

	var peerConn = accept(listener, leechAddr);
	defer peerConn.Close();
	listener.Close();

	var seedConn = dial(seedAddr);
	defer seedConn.Close();

	transfer(io.MultiWriter(outFile, peerConn), seedConn);
}

func leech(transferFileName, seedAddr string){
	outFile, err := os.Create(transferFileName);
	if err != nil { log.Fatal("Error create file: ", err.Error()); }
	defer flushFile(outFile);

	var seedConn = dial(seedAddr);
	defer seedConn.Close();

	transfer(outFile, seedConn);
}

func main(){
	var mode       = flag.String("mode",  "",  "seed | peer | leech");
	var listenPort = flag.String("port",  "",  "Listen port");
	var leechAddr  = flag.String("leech", "",  "Leech IP");
	var seedAddr   = flag.String("seed",  "",  "Seed IP:port. Example: 192.168.0.1:1234");
	var inputFile  = flag.String("if",    "",  "Input file path");
	var outputFile = flag.String("of",    "",  "Output file path");

	flag.Parse();

	var Fatal = func(message ...interface{}){
		fmt.Fprintln(os.Stderr, message);
		flag.Usage();
		os.Exit(1);
	}

	switch(*mode){
	case "seed":
		if *inputFile=="" || *listenPort=="" || *leechAddr=="" {
			Fatal(`Flags "if", "port", "leech" - must be present.`);
		}
		seed(*inputFile, *listenPort, *leechAddr);

	case "peer":
		if *outputFile=="" || *listenPort=="" || *seedAddr=="" || *leechAddr=="" {
			Fatal(`Flags "of", "port", "seed", "leech" - must be present.`);
		}
		peer(*outputFile, *listenPort, *seedAddr, *leechAddr);

	case "leech":
		if *outputFile=="" || *seedAddr=="" {
			Fatal(`Flags "of", "seed" - must be present.`);
		}
		leech(*outputFile, *seedAddr);

	default:
		Fatal("Incorrect mode - ", *mode);
	}

}
