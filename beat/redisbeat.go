package beat

import (
    "errors"
    "regexp"
    "strconv"
    "strings"
    "time"

    "github.com/elastic/beats/libbeat/beat"
    "github.com/elastic/beats/libbeat/cfgfile"
    "github.com/elastic/beats/libbeat/common"
    "github.com/elastic/beats/libbeat/logp"
    "github.com/elastic/beats/libbeat/publisher"
    "github.com/garyburd/redigo/redis"
)

type Redisbeat struct {
    period           time.Duration
    host             string
    port             int
    network          string
    maxConn          int
    auth             bool
    pass             string
    RbConfig         ConfigSettings
    events           publisher.Client

    serverStats      bool
    clientsStats     bool
    memoryStats      bool
    persistenceStats bool
    statsStats       bool
    replicationStats bool
    cpuStats         bool
    commandStats     bool
    clusterStats     bool
    keyspaceStats    bool
    keyPattern       []string
    keyPatternLimit  int
    redisBeatCacheKey string
    cacheExpiryTime  int

    redisPool        *redis.Pool
    done             chan struct{}
}

func New() *Redisbeat {
    return &Redisbeat{}
}

func (rb *Redisbeat) Config(b *beat.Beat) error {
    err := cfgfile.Read(&rb.RbConfig, "")
    if err != nil {
        logp.Err("Error reading configuration file: %v", err)
        return err
    }

    if rb.RbConfig.Input.Period != nil {
        rb.period = time.Duration(*rb.RbConfig.Input.Period) * time.Second
    } else {
        rb.period = DEFAULT_PERIOD
    }

    if rb.RbConfig.Input.Host != nil {
        rb.host = *rb.RbConfig.Input.Host
    } else {
        rb.host = DEFAULT_HOST
    }

    if rb.RbConfig.Input.Port != nil {
        rb.port = *rb.RbConfig.Input.Port
    } else {
        rb.port = DEFAULT_PORT
    }

    if rb.RbConfig.Input.Network != nil {
        rb.network = *rb.RbConfig.Input.Network
    } else {
        rb.network = DEFAULT_NETWORK
    }

    if rb.RbConfig.Input.MaxConn != nil {
        rb.maxConn = *rb.RbConfig.Input.MaxConn
    } else {
        rb.maxConn = DEFAULT_MAX_CONN
    }

    if rb.RbConfig.Input.Auth.Required != nil {
        rb.auth = *rb.RbConfig.Input.Auth.Required
    } else {
        rb.auth = DEFAULT_AUTH_REQUIRED
    }

    if rb.RbConfig.Input.Auth.RequiredPass != nil {
        rb.pass = *rb.RbConfig.Input.Auth.RequiredPass
    } else {
        rb.pass = DEFAULT_AUTH_REQUIRED_PASS
    }

    if rb.RbConfig.Input.Stats.Server != nil {
        rb.serverStats = *rb.RbConfig.Input.Stats.Server
    } else {
        rb.serverStats = DEFAULT_STATS_SERVER
    }

    if rb.RbConfig.Input.Stats.Clients != nil {
        rb.clientsStats = *rb.RbConfig.Input.Stats.Clients
    } else {
        rb.clientsStats = DEFAULT_STATS_CLIENT
    }

    if rb.RbConfig.Input.Stats.Memory != nil {
        rb.memoryStats = *rb.RbConfig.Input.Stats.Memory
    } else {
        rb.memoryStats = DEFAULT_STATS_MEMORY
    }

    if rb.RbConfig.Input.Stats.Persistence != nil {
        rb.persistenceStats = *rb.RbConfig.Input.Stats.Persistence
    } else {
        rb.persistenceStats = DEFAULT_STATS_PERSISTENCE
    }

    if rb.RbConfig.Input.Stats.Stats != nil {
        rb.statsStats = *rb.RbConfig.Input.Stats.Stats
    } else {
        rb.statsStats = DEFAULT_STATS_STATS
    }

    if rb.RbConfig.Input.Stats.Replication != nil {
        rb.replicationStats = *rb.RbConfig.Input.Stats.Replication
    } else {
        rb.replicationStats = DEFAULT_STATS_REPLICATION
    }

    if rb.RbConfig.Input.Stats.Cpu != nil {
        rb.cpuStats = *rb.RbConfig.Input.Stats.Cpu
    } else {
        rb.cpuStats = DEFAULT_STATS_CPU
    }

    if rb.RbConfig.Input.Stats.Commandstats != nil {
        rb.commandStats = *rb.RbConfig.Input.Stats.Commandstats
    } else {
        rb.commandStats = DEFAULT_STATS_COMMAND
    }

    if rb.RbConfig.Input.Stats.Cluster != nil {
        rb.clusterStats = *rb.RbConfig.Input.Stats.Cluster
    } else {
        rb.clusterStats = DEFAULT_STATS_CLUSTER
    }

    if rb.RbConfig.Input.Stats.Keyspace != nil {
        rb.keyspaceStats = *rb.RbConfig.Input.Stats.Keyspace
    } else {
        rb.keyspaceStats = DEFAULT_STATS_KEYSPACE
    }
    if rb.RbConfig.Input.Stats.KeyPattern != nil {
        rb.keyPattern = *rb.RbConfig.Input.Stats.KeyPattern
    } else {
        rb.keyPattern = DEFAULT_KEY_PATTERN
    }
    if rb.RbConfig.Input.RedisBeatCacheKey != nil{
        rb.redisBeatCacheKey = *rb.RbConfig.Input.RedisBeatCacheKey
    }else{
       rb.redisBeatCacheKey = DEFAULT_REDIS_BEAT_CACHE_KEY
    }
    if rb.RbConfig.Input.KeyPatternLimit != nil {
        rb.keyPatternLimit = *rb.RbConfig.Input.KeyPatternLimit
    }else{
       rb.keyPatternLimit = DEFAULT_KEY_PATTERN_LIMIT
    }
    if rb.RbConfig.Input.CacheExpiryTime !=nil {
        rb.cacheExpiryTime = *rb.RbConfig.Input.CacheExpiryTime
    }else{
        rb.cacheExpiryTime = DEFAULT_CACHE_EXPIRY_TIME
    }

    logp.Debug("redisbeat", "Init redisbeat")
    logp.Debug("redisbeat", "Period %v\n", rb.period)
    logp.Debug("redisbeat", "Host %v\n", rb.host)
    logp.Debug("redisbeat", "Port %v\n", rb.port)
    logp.Debug("redisbeat", "Network %v\n", rb.network)
    logp.Debug("redisbeat", "Max Connections %v\n", rb.maxConn)
    logp.Debug("redisbeat", "Auth %t\n", rb.auth)
    logp.Debug("redisbeat", "Server statistics %t\n", rb.serverStats)
    logp.Debug("redisbeat", "Client statistics %t\n", rb.clientsStats)
    logp.Debug("redisbeat", "Memory statistics %t\n", rb.memoryStats)
    logp.Debug("redisbeat", "Persistence statistics %t\n", rb.persistenceStats)
    logp.Debug("redisbeat", "Stats statistics %t\n", rb.statsStats)
    logp.Debug("redisbeat", "Replication statistics %t\n", rb.replicationStats)
    logp.Debug("redisbeat", "Cpu statistics %t\n", rb.cpuStats)
    logp.Debug("redisbeat", "Command statistics %t\n", rb.commandStats)
    logp.Debug("redisbeat", "Cluster statistics %t\n", rb.clusterStats)
    logp.Debug("redisbeat", "Keyspace statistics %t\n", rb.keyspaceStats)
    logp.Debug("redisbeat", "KeyPatterns to be fetched %v\n", rb.keyPattern)
    logp.Debug("redisbeat", "Cache Key set to %v\n",rb.redisBeatCacheKey)
    logp.Debug("redisbeat", "Key Pattern Limit set to %v\n", rb.keyPatternLimit)
    logp.Debug("redisbeat", "Key Pattern Limit set to %v\n", rb.cacheExpiryTime)
    return nil
}

func (rb *Redisbeat) Setup(b *beat.Beat) error {
    rb.events = b.Events
    rb.done = make(chan struct{})

    // Set up redis pool
    redisPool := redis.NewPool(func() (redis.Conn, error) {
        c, err := redis.Dial(rb.network, rb.host + ":" + strconv.Itoa(rb.port))

        if err != nil {
            return nil, err
        }

        return c, err
    }, rb.maxConn)

    rb.redisPool = redisPool

    if rb.auth {
        c := rb.redisPool.Get()
        defer c.Close()

        authed, err := c.Do("AUTH", rb.pass)
        if err != nil {
            return err
        } else {
            logp.Debug("redisbeat", "AUTH %v", authed)
        }
    }

    return nil
}

func (r *Redisbeat) Run(b *beat.Beat) error {
    var err error

    ticker := time.NewTicker(r.period)
    defer ticker.Stop()

    for {
        select {
        case <-r.done:
            return nil
        case <-ticker.C:
        }

        timerStart := time.Now()
        r.exportKeys()
        if r.serverStats {

            err = r.exportStats("server")
            if err != nil {
                logp.Err("Error reading server stats: %v", err)
                break
            }
        }
        if r.clientsStats {
            err = r.exportStats("clients")
            if err != nil {
                logp.Err("Error reading clients stats: %v", err)
                break
            }
        }
        if r.memoryStats {
            err = r.exportStats("memory")
            if err != nil {
                logp.Err("Error reading memory stats: %v", err)
                break
            }
        }
        if r.persistenceStats {
            err = r.exportStats("persistence")
            if err != nil {
                logp.Err("Error reading persistence stats: %v", err)
                break
            }
        }
        if r.statsStats {
            err = r.exportStats("stats")
            if err != nil {
                logp.Err("Error reading stats stats: %v", err)
                break
            }
        }
        if r.replicationStats {
            err = r.exportStats("replication")
            if err != nil {
                logp.Err("Error reading replication stats: %v", err)
                break
            }
        }
        if r.cpuStats {
            err = r.exportStats("cpu")
            if err != nil {
                logp.Err("Error reading cpu stats: %v", err)
                break
            }
        }
        if r.commandStats {
            err = r.exportStats("commandstats")
            if err != nil {
                logp.Err("Error reading commandstats: %v", err)
                break
            }
        }
        if r.clusterStats {
            err = r.exportStats("cluster")
            if err != nil {
                logp.Err("Error reading cluster stats: %v", err)
                break
            }
        }
        if r.keyspaceStats {
            err = r.exportStats("keyspace")
            if err != nil {
                logp.Err("Error reading keypsace stats: %v", err)
                break
            }
        }

        timerEnd := time.Now()
        duration := timerEnd.Sub(timerStart)
        if duration.Nanoseconds() > r.period.Nanoseconds() {
            logp.Warn("Ignoring tick(s) due to processing taking longer than one period")
        }
    }

    return err
}

func (rb *Redisbeat) Cleanup(b *beat.Beat) error {
    // I wonder if the redis pool should released here, after the main loop exists.
    return nil
}

// Stop is triggered on exit, closing the done channel and redis pool
func (r *Redisbeat) Stop() {
    close(r.done)
    r.redisPool.Close()
}

func (r *Redisbeat) exportStats(statType string) error {
    stats, err := r.getInfoReply(statType)
    if err != nil {
        logp.Warn("Failed to fetch server stats: %v", err)
        return err
    }

    event := common.MapStr{
        "@timestamp": common.Time(time.Now()),
        "type":       statType,
        "count":      1,
        "stats":      stats,
    }

    r.events.PublishEvent(event)

    return nil
}

// getInfoReply sends INFO type command and returns the response as a map
func (r *Redisbeat) getInfoReply(infoType string) (map[string]string, error) {
    c := r.redisPool.Get()
    defer c.Close()
    reply, err := redis.Bytes(c.Do("INFO", infoType))

    if err != nil {
        return nil, err
    } else {
        s := string(reply[:])
        return convertReplyToMap(s)
    }
}
func keysCacheExist(r *Redisbeat) bool{
    c := r.redisPool.Get()
    defer c.Close()
    res, err := redis.Bool(c.Do("EXISTS", r.redisBeatCacheKey))
    if err != nil {
        logp.Debug("error while checking the existance of key cache", err.Error())
    }
    return res
}

func (r *Redisbeat) generateKeysCache() {
    keyTypeMap := make(map[string][]interface{})
    for _,patt:= range r.keyPattern{
        tempMap,err := r.getKeysAndTypeMap(patt)
        if err == redis.ErrNil{
            logp.Debug("No keys found for Pattern %s", patt)
            return
        }else if err != nil{
            logp.Debug("Error occured whilr trying to get keys matching pattern %s",patt,err.Error())
            return
        }
        for key,keytype := range tempMap{
                if SUPPORTED_TYPES[keytype]{
                    keyTypeMap[keytype]= append(keyTypeMap[keytype],key)
                }
            }
    }
    r.cacheKeyPatterns(keyTypeMap)
    keyslist := []interface{}{"gaurav","das","das"}
    r.setCacheKey(r.redisBeatCacheKey,keyslist)
}

func (r *Redisbeat) cacheKeyPatterns(keyTypeMap map[string][]interface{}){
    for key,_ := range SUPPORTED_TYPES{
        listname := r.redisBeatCacheKey+key
        if keyTypeMap[key] != nil{
            r.setCacheKey(listname,keyTypeMap[key])
        }
    }
}


func (r *Redisbeat) setCacheKey(key string, value []interface{}){
    c := r.redisPool.Get()
    defer c.Close()
    _,er := c.Do("RPUSH",append([]interface{}{key}, value...)...)
    if er != nil {
        logp.Err("Could not add cache key %v",key,er.Error())
    } else {
        res,err := redis.Bool(c.Do("EXPIRE",append([]interface{}{key},r.cacheExpiryTime)...))
        if err != nil || res ==false{
            logp.Err("Could not set Expire time on key %v",key,err.Error())
        }
    }
}

func (r *Redisbeat) getKeysAndTypeMap(pattern string) (map[string]string, error) {
    c := r.redisPool.Get()
    defer c.Close()
    strMap := map[string]string{}
    keys, err := redis.Strings(c.Do("keys", pattern))
    if err != nil {
        return nil, err
    }
    for _, k := range keys {
        keytype, err := redis.String(c.Do("TYPE", k))
        if err != nil {
        logp.Debug("Error while fetching type for key = %s, err = %v", k, err.Error())
        }
        strMap[k] = keytype
    }
    return strMap,nil
}

func (r *Redisbeat) publishEvent(eventMap map[string]interface{},eventType string) error{
    event := common.MapStr{
                "@timestamp": common.Time(time.Now()),
                "type":      eventType,
        }
    for k, v := range eventMap {
        event[k] = v
    }
    r.events.PublishEvent(event)
    return nil
}

func (r *Redisbeat) exportStringKeys() error{
    c := r.redisPool.Get()
    defer c.Close()
    listname := r.redisBeatCacheKey + "string"
    keylist, err := redis.Strings(c.Do("LRANGE",listname,0,r.keyPatternLimit))
    if err != nil {
        logp.Debug("Could not fetch list of string Keys",err.Error())
        return nil
    }else{
        eventMap := r.getStringKeys(keylist)
        r.publishEvent(eventMap,"string_keys")
    }
    return nil
}

func (r *Redisbeat) exportHashKeys() error{
    c := r.redisPool.Get()
    defer c.Close()
    listname := r.redisBeatCacheKey + "hash"
    keylist, err := redis.Strings(c.Do("LRANGE",listname,0,r.keyPatternLimit))
    if err != nil {
        logp.Debug("Could not fetch list of hash Keys",err.Error())
        return nil
    }else{
        r.getAndPublishHashKeys(keylist)
    }
    return nil
}

func (r *Redisbeat) getAndPublishHashKeys(keyslist []string) (map[string]interface{}) {
    c := r.redisPool.Get()
    defer c.Close()
    for _,key := range keyslist{
        values, err := redis.StringMap(c.Do("hgetall",key))
        if err != nil {
            logp.Debug("Error while getting values", err.Error())
        } else {
            r.publishHashKey(values)
        }
    }
    return nil
}

func (r *Redisbeat) publishHashKey(eventMap map[string]string) error{
    event := common.MapStr{
                "@timestamp": common.Time(time.Now()),
                "type":      "hash_key",
        }
    for k, v := range eventMap {
        event[k] = v
    }
    r.events.PublishEvent(event)
    return nil
}

func (r *Redisbeat) getListKeys(keyslist []string) (map[string]interface{}) {
    c := r.redisPool.Get()
    defer c.Close()
    strMap := make(map[string]interface{})
    for _,key := range keyslist{
        values, err := redis.Strings(c.Do("LRANGE",key, 0, -1))
        if err != nil {
            logp.Debug("Error while getting values", err.Error())
        } else {
            strMap[key] = values
        }
    }
    return strMap
}

func (r *Redisbeat) exportListKeys() error{
    c := r.redisPool.Get()
    defer c.Close()
    listname := r.redisBeatCacheKey + "list"
    keylist, err := redis.Strings(c.Do("LRANGE",listname,0,r.keyPatternLimit))
    if err != nil {
        logp.Debug("Could not fetch list of List Keys",err.Error())
        return nil
    }else{
        eventMap := r.getListKeys(keylist)
        r.publishEvent(eventMap,"list_keys")
    }
    return nil
}

func (r *Redisbeat) exportKeys() error {
    if keysCacheExist(r) == false {
        r.generateKeysCache()
    }
    r.exportStringKeys()
    r.exportHashKeys()
    r.exportListKeys()
    return nil
}

func (r *Redisbeat) getStringKeys(keyslist []string) (map[string]interface{}) {
    c := r.redisPool.Get()
    defer c.Close()
    var args []interface{}
    for _,key := range keyslist{
        args = append(args,key)
    }
    values, err := redis.Strings(c.Do("MGET", args...))
    if err != nil {
        logp.Debug("Error while getting values", err.Error())
        return nil
    }
    strMap := make(map[string]interface{})
    for i, v := range values {
        strMap[keyslist[i]] = v
    }
    return strMap

}

// convertReplyToMap converts a bulk string reply from Redis to a map
func convertReplyToMap(s string) (map[string]string, error) {
    var info map[string]string
    info = make(map[string]string)

    // Regex for INFO type property
    infoRegex := `^\s*#\s*\w+\s*$`
    r, err := regexp.Compile(infoRegex)
    if err != nil {
        return nil, errors.New("Regex failed to compile")
    }

    // http://redis.io/topics/protocol#bulk-string-reply
    a := strings.Split(s, "\r\n")

    for _, v := range a {
        if r.MatchString(v) || v == "" {
            logp.Debug("redisbeat", "Skipping reply string - \"%v\"", v)
            continue
        }
        entry := strings.Split(v, ":")
        logp.Debug("redisbeat", "Entry: %#v\n", entry)
        info[entry[0]] = entry[1]
    }
    return info, nil
}
