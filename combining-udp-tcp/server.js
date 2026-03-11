const http = require('http');

const PORT = process.env.PORT || 3000;
const MESSAGE = process.env.MESSAGE || 'Hello from server';

const server = http.createServer((req, res) => {
  if (req.url === '/hello' && req.method === 'GET') {
    res.writeHead(200, { 'Content-Type': 'text/plain' });
    res.end(MESSAGE);
  } else {
    res.writeHead(404, { 'Content-Type': 'text/plain' });
    res.end('Not Found');
  }
});

server.listen(PORT, () => {
  console.log(`Server running on http://localhost:${PORT}`);
  console.log(`Message: "${MESSAGE}"`);
});
