const start = Date.now();
console.log('Ingestion started at: '+JSON.stringify(Math.floor(start / 1000)));
//console.log('Ingestion started at: '+JSON.stringify(new Date(start)));

var values = []
for (let i = 5; i >= 0; i--) {
    values.push({
        "timestamp": new Date(start-i*60000),
        "cpu_load": getRandomFloat(0,100,2),
        "concurrency": getRandomInt(0, 500000)
    });
}

//console.log("Values :"+ JSON.stringify(values, null, 2));
console.log("Inserting documents to MongoDB");
db = db.getSiblingDB('sky')
db.metrics.insertMany(values)

console.log("Done");

function getRandomFloat(min, max, decimals) {
    const str = (Math.random() * (max - min) + min).toFixed(decimals);
    return parseFloat(str);
}

function getRandomInt(min, max) {
    return Math.floor(Math.random() * (max - min) + min);
}