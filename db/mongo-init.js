db.createUser({
    user: 'user',
    pwd: 'password',
    roles: [
      {
        role: 'dbOwner',
        db: 'sky',
      },
    ],
  });

db.createCollection("metrics", {
    timeseries: {
      timeField: "timestamp",
      granularity: "minutes"
      // metaField: "cpu/load-concurrency",
    },
    expireAfterSeconds: 315360000 // 10 years 
  }); 

db.metrics.insertMany([
  {
    "timestamp": new Date(1501685060000),
    "cpu_load": 48,
    "concurrency": 365984
  },
  {
    "timestamp": new Date(1501685120000),
    "cpu_load": 66,
    "concurrency": 125847
  },
  {
    "timestamp": new Date(1501685180000),
    "cpu_load": 55,
    "concurrency": 500000
  },
  {
    "timestamp": new Date(1501685240000),
    "cpu_load": 100,
    "concurrency": 5
  },
  {
    "timestamp": new Date(1501685300000),
    "cpu_load": 50,
    "concurrency": 12589
  },
  {
    "timestamp": new Date(1501685360000),
    "cpu_load": 1,
    "concurrency": 10000
  }]);
