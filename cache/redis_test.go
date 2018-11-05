package cache

import (
	"fmt"
	"github.com/go-redis/redis"
	"testing"
	"time"
)

func Test1(t *testing.T) {

	client := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:11511",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	err := client.Set("name", "redis", 0).Err()
	if err != nil {
		t.Fatal(err)
	}

	val, err := client.Get("name").Result()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("name", val)

	val2, err := client.Get("key2").Result()
	if err == redis.Nil {
		fmt.Println("key2 does not exists")
	} else if err != nil {
		t.Fatal(err)
	} else {
		fmt.Println("key2", val2)
	}

	del, err := client.Del("hello").Result()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(del)
}

func Test2(t *testing.T) {
	DefaultRedisManager().Register("test", "127.0.0.1:11511", "", 0)
	client, _ := DefaultRedisManager().Get("test")
	err := client.Set("tommy", "tommy", 0).Err()
	if err != nil {
		t.Fatal(err)
	}

	val, err := client.Get("tommy").Result()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("tommy", val)
}


func TestString(t *testing.T) {

	client := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:11511",
		Password: "", // no password set
		DB:       0,  // use default DB
		DialTimeout:time.Second*3,
	})
	err := client.Set("name", "redis", 0).Err()
	if err != nil {
		fmt.Println(err)
		return
	}
	//return a string
	val, err := client.Get("name").Result()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("name:", val)


}

func TestHash(t *testing.T) {

	client := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:11511",
		Password: "", // no password set
		DB:       0,  // use default DB
		DialTimeout:time.Second*3,
	})
	err := client.HSet("k1", "f1", "v1").Err()
	if err != nil {
		fmt.Println(err)
		return
	}

	//return a string
	val, err := client.HGet("k1","f1").Result()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("k1","f1:", val)
	fields:=make(map[string]interface{})
	fields["f2"]="v2"
	fields["f3"]=3
	err = client.HMSet("k1", fields).Err()
	if err != nil {
		fmt.Println(err)
		return
	}
	//return a map[string]string
	multiVal, err := client.HGetAll("k1").Result()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("k1:", multiVal)
	client.Close()

}

