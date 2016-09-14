package main

import (
	"testing"
	"flag"
	"os"
)

var (
	seedFile  *string;
	peerFile  *string;
	leechFile *string;
)

func init(){
	seedFile  = flag.String("seed",  "", "Seed file path (In)");
	peerFile  = flag.String("peer",  "", "Peer file path (Out)");
	leechFile = flag.String("leech", "", "Leech file path (Out)");
}

// go test -seed="..." -peer="..." -leech="..."
func TestMain(m *testing.M){
	if !flag.Parsed() {
		flag.Parse();
	}

	os.Exit(m.Run());
}

func TestStand(t *testing.T){
	if *seedFile == "" || *peerFile == "" || *leechFile == "" {
		t.Skip(`Please set all files in command line: -seed="..." -peer="..." -leech="..."`);
	}

	var c = make(chan int);

	go func(){ defer func(){ c<-1 }(); seed(  *seedFile, "56001",                    "127.0.0.1"); }();
	go func(){ defer func(){ c<-1 }(); peer(  *peerFile, "56002", "127.0.0.1:56001", "127.0.0.1"); }();
	go func(){ defer func(){ c<-1 }(); leech(*leechFile,          "127.0.0.1:56002"             ); }();

	<-c;<-c;<-c;
}