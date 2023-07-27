package main

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestLimit(t *testing.T) {
	maxToken = 220
	client, err := getClient(token, "")
	if err != nil {
		panic(err)
	}
	w := NewWriter(os.Stdout.Write, nil, nil)
	err = client.stream(context.TODO(), "我需要一个golang 代码能够下载文件", w)
	if err != nil {
		panic(err)
	}
	err = client.stream(context.TODO(), "这段代码里面的filepath 是什么意思呢", w)
	if err != nil {
		panic(err)
	}
}

func TestExpired(t *testing.T) {
	client, err := getClient(token, "")
	if err != nil {
		panic(err)
	}
	defaultExpire = time.Second
	w := NewWriter(os.Stdout.Write, nil, nil)
	err = client.stream(context.TODO(), "玩一个游戏，我说hello，你只能说hi", w)
	if err != nil {
		panic(err)
	}
	time.Sleep(defaultExpire)
	err = client.stream(context.TODO(), "hello", w)
	if err != nil {
		panic(err)
	}
}

func Test1(t *testing.T) {
	client, err := getClient(token, "")
	if err != nil {
		panic(err)
	}
	w := NewWriter(os.Stdout.Write, nil, nil)
	err = client.stream(context.TODO(), "玩一个游戏，我说hello，你只能说hi", w)
	if err != nil {
		panic(err)
	}
	err = client.stream(context.TODO(), "hello", w)
	if err != nil {
		panic(err)
	}
}
