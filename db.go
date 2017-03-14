package main

import (
	"github.com/garyburd/redigo/redis"
	"github.com/pkg/errors"
)

type DB interface {
	get(field string) (string, error)
	close() error
}

type Redis struct {
	conn    redis.Conn
	hashKey string
}

func newRedis(host, port, defaultDestURL, hashKey string) (DB, error) {
	c, err := redis.Dial("tcp", host+":"+port)
	if err != nil {
		return nil, errors.Wrap(err, "Failed start connection to Redis")
	}
	_, err = c.Do("HSET", hashKey, "default", defaultDestURL)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed Redis Command 'HSET %s default %s'", hashKey, defaultDestURL)
	}
	r := &Redis{
		conn:    c,
		hashKey: hashKey,
	}
	return r, nil
}

// If the field does not exist, then the return value is the value of the "default" key
func (r *Redis) get(field string) (string, error) {
	reply, err := redis.String(r.conn.Do("HGET", r.hashKey, field))
	switch {
	case err == redis.ErrNil: // when the field not exist field
		reply, _ = redis.String(r.conn.Do("HGET", r.hashKey, "default"))
	case err != nil:
		return "", errors.Wrapf(err, "Failed Redis Command 'HGET %s %s'", r.hashKey, field)
	}
	return reply, nil
}

func (r *Redis) close() error {
	err := r.conn.Close()
	if err != nil {
		return errors.Wrap(err, "Failed close connection to Redis")
	}
	return nil
}
