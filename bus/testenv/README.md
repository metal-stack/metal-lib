# Create required self-signed certificate with custom root CA

Run the following commands from within `./certs` subdirectory:

```
cfssl genkey -initca csr.json | cfssljson -bare ca_cert && \
cfssl gencert -ca ca_cert.pem -ca-key ca_cert-key.pem csr.json | cfssljson -bare client_cert && \
cat client_cert.pem client_cert-key.pem > client.pem
```

# Test with docker-compose

```
make
```

# Test with cURL
```
docker-compose up -d --build
```

Download server certificate (only once the first time):

```
echo quit | openssl s_client -showcerts -connect localhost:4152 > server_cert.pem
```

We can now connect to NSQD and create a test topic:

```
curl -v -X POST --key client_key.pem --cert client_cert.pem --cacert server_cert.pem --resolve metal-control-plane-nsqd:4152:127.0.0.1 https://metal-control-plane-nsqd:4152/topic/create?topic=test
```

Verifiy with

```
curl -v --key client_key.pem --cert client_cert.pem --cacert server_cert.pem --resolve metal-control-plane-nsqd:4152:127.0.0.1 https://metal-control-plane-nsqd:4152/stats?format=json&topic=test
```

or through NSQ-Admin website:

```
http://localhost:4171/
```
