# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

CRT_LOCATION = certs
O = Supermq
OU_CA = supermq_ca
OU_CRT = supermq_crt
EA = info@ultraviolet.rs
CN_CA = Supermq_Self_Signed_CA
CN_SRV = localhost
THING_SECRET = <THING_SECRET> # e.g. 8f65ed04-0770-4ce4-a291-6d1bf2000f4d
CRT_FILE_NAME = thing
THINGS_GRPC_SERVER_CONF_FILE_NAME=thing-grpc-server.conf
THINGS_GRPC_CLIENT_CONF_FILE_NAME=thing-grpc-client.conf
THINGS_GRPC_SERVER_CN=things
THINGS_GRPC_CLIENT_CN=things-client
THINGS_GRPC_SERVER_CRT_FILE_NAME=things-grpc-server
THINGS_GRPC_CLIENT_CRT_FILE_NAME=things-grpc-client
AUTH_GRPC_SERVER_CONF_FILE_NAME=auth-grpc-server.conf
AUTH_GRPC_CLIENT_CONF_FILE_NAME=auth-grpc-client.conf
AUTH_GRPC_SERVER_CN=auth
AUTH_GRPC_CLIENT_CN=auth-client
AUTH_GRPC_SERVER_CRT_FILE_NAME=auth-grpc-server
AUTH_GRPC_CLIENT_CRT_FILE_NAME=auth-grpc-client
AGENT_CRT_LOCATION = agent
AGENT_CA_CN = Cube_Agent_CA
AGENT_SERVER_CN = cube-agent
AGENT_CLIENT_CN = cube-agent-client

define GRPC_CERT_CONFIG
[req]
req_extensions = v3_req
distinguished_name = dn
prompt = no

[dn]
CN = mg.svc
C  = RS
ST = RS
L  = BELGRADE
O  = SUPERMQ
OU = SUPERMQ

[v3_req]
subjectAltName = @alt_names

[alt_names]
DNS.1 = <<SERVICE_NAME>>
endef

define AGENT_CERT_CONFIG
[req]
req_extensions = v3_req
distinguished_name = dn
prompt = no

[dn]
CN = <<SERVICE_NAME>>
C  = RS
ST = RS
L  = BELGRADE
O  = SUPERMQ
OU = SUPERMQ

[v3_req]
subjectAltName = @alt_names

[alt_names]
DNS.1 = <<SERVICE_NAME>>
DNS.2 = localhost
endef

define ANNOUNCE_BODY
Version $(VERSION) of $(PACKAGE_NAME) has been released.

It can be downloaded from $(DOWNLOAD_URL).

etc, etc.
endef
all: clean_certs ca server_cert things_grpc_certs auth_grpc_certs agent_certs

# CA name and key is "ca".
ca:
	openssl req -newkey rsa:2048 -x509 -nodes -sha512 -days 1095 \
				-keyout $(CRT_LOCATION)/ca.key -out $(CRT_LOCATION)/ca.crt -subj "/CN=$(CN_CA)/O=$(O)/OU=$(OU_CA)/emailAddress=$(EA)"

# Server cert and key name is "supermq-server".
server_cert:
	# Create supermq server key and CSR.
	openssl req -new -sha256 -newkey rsa:4096 -nodes -keyout $(CRT_LOCATION)/supermq-server.key \
				-out $(CRT_LOCATION)/supermq-server.csr -subj "/CN=$(CN_SRV)/O=$(O)/OU=$(OU_CRT)/emailAddress=$(EA)"

	# Sign server CSR.
	openssl x509 -req -days 1000 -in $(CRT_LOCATION)/supermq-server.csr -CA $(CRT_LOCATION)/ca.crt -CAkey $(CRT_LOCATION)/ca.key -CAcreateserial -out $(CRT_LOCATION)/supermq-server.crt

	# Remove CSR.
	rm $(CRT_LOCATION)/supermq-server.csr

thing_cert:
	# Create supermq server key and CSR.
	openssl req -new -sha256 -newkey rsa:4096 -nodes -keyout $(CRT_LOCATION)/$(CRT_FILE_NAME).key \
				-out $(CRT_LOCATION)/$(CRT_FILE_NAME).csr -subj "/CN=$(THING_SECRET)/O=$(O)/OU=$(OU_CRT)/emailAddress=$(EA)"

	# Sign client CSR.
	openssl x509 -req -days 730 -in $(CRT_LOCATION)/$(CRT_FILE_NAME).csr -CA $(CRT_LOCATION)/ca.crt -CAkey $(CRT_LOCATION)/ca.key -CAcreateserial -out $(CRT_LOCATION)/$(CRT_FILE_NAME).crt

	# Remove CSR.
	rm $(CRT_LOCATION)/$(CRT_FILE_NAME).csr

things_grpc_certs:
	# Things server grpc certificates
	$(file > $(CRT_LOCATION)/$(THINGS_GRPC_SERVER_CRT_FILE_NAME).conf,$(subst <<SERVICE_NAME>>,$(THINGS_GRPC_SERVER_CN),$(GRPC_CERT_CONFIG)) )

	openssl req -new -sha256  -newkey rsa:4096 -nodes \
				-keyout $(CRT_LOCATION)/$(THINGS_GRPC_SERVER_CRT_FILE_NAME).key  \
				-out $(CRT_LOCATION)/$(THINGS_GRPC_SERVER_CRT_FILE_NAME).csr \
				-config $(CRT_LOCATION)/$(THINGS_GRPC_SERVER_CRT_FILE_NAME).conf \
				-extensions v3_req

	openssl x509 -req -sha256 \
			-in $(CRT_LOCATION)/$(THINGS_GRPC_SERVER_CRT_FILE_NAME).csr \
			-CA $(CRT_LOCATION)/ca.crt \
			-CAkey $(CRT_LOCATION)/ca.key \
			-CAcreateserial \
			-out $(CRT_LOCATION)/$(THINGS_GRPC_SERVER_CRT_FILE_NAME).crt \
			-days 365 \
			-extfile $(CRT_LOCATION)/$(THINGS_GRPC_SERVER_CRT_FILE_NAME).conf \
			-extensions v3_req

	rm -rf  $(CRT_LOCATION)/$(THINGS_GRPC_SERVER_CRT_FILE_NAME).csr $(CRT_LOCATION)/$(THINGS_GRPC_SERVER_CRT_FILE_NAME).conf
	# Things client grpc certificates
	$(file > $(CRT_LOCATION)/$(THINGS_GRPC_CLIENT_CRT_FILE_NAME).conf,$(subst <<SERVICE_NAME>>,$(THINGS_GRPC_CLIENT_CN),$(GRPC_CERT_CONFIG)) )

	openssl req -new -sha256 -newkey rsa:4096 -nodes \
				-keyout $(CRT_LOCATION)/$(THINGS_GRPC_CLIENT_CRT_FILE_NAME).key  \
				-out $(CRT_LOCATION)/$(THINGS_GRPC_CLIENT_CRT_FILE_NAME).csr \
				-config $(CRT_LOCATION)/$(THINGS_GRPC_CLIENT_CRT_FILE_NAME).conf \
				-extensions v3_req

	openssl x509 -req -sha256 \
			-in $(CRT_LOCATION)/$(THINGS_GRPC_CLIENT_CRT_FILE_NAME).csr \
			-CA $(CRT_LOCATION)/ca.crt \
			-CAkey $(CRT_LOCATION)/ca.key \
			-CAcreateserial \
			-out $(CRT_LOCATION)/$(THINGS_GRPC_CLIENT_CRT_FILE_NAME).crt \
			-days 365 \
			-extfile $(CRT_LOCATION)/$(THINGS_GRPC_CLIENT_CRT_FILE_NAME).conf \
			-extensions v3_req

	rm -rf  $(CRT_LOCATION)/$(THINGS_GRPC_CLIENT_CRT_FILE_NAME).csr $(CRT_LOCATION)/$(THINGS_GRPC_CLIENT_CRT_FILE_NAME).conf

auth_grpc_certs:
	# Auth gRPC server certificate
	$(file > $(CRT_LOCATION)/$(AUTH_GRPC_SERVER_CRT_FILE_NAME).conf,$(subst <<SERVICE_NAME>>,$(AUTH_GRPC_SERVER_CN),$(GRPC_CERT_CONFIG)) )

	openssl req -new -sha256  -newkey rsa:4096 -nodes \
				-keyout $(CRT_LOCATION)/$(AUTH_GRPC_SERVER_CRT_FILE_NAME).key  \
				-out $(CRT_LOCATION)/$(AUTH_GRPC_SERVER_CRT_FILE_NAME).csr \
				-config $(CRT_LOCATION)/$(AUTH_GRPC_SERVER_CRT_FILE_NAME).conf \
				-extensions v3_req

	openssl x509 -req -sha256 \
			-in $(CRT_LOCATION)/$(AUTH_GRPC_SERVER_CRT_FILE_NAME).csr \
			-CA $(CRT_LOCATION)/ca.crt \
			-CAkey $(CRT_LOCATION)/ca.key \
			-CAcreateserial \
			-out $(CRT_LOCATION)/$(AUTH_GRPC_SERVER_CRT_FILE_NAME).crt \
			-days 365 \
			-extfile $(CRT_LOCATION)/$(AUTH_GRPC_SERVER_CRT_FILE_NAME).conf \
			-extensions v3_req

	rm -rf  $(CRT_LOCATION)/$(AUTH_GRPC_SERVER_CRT_FILE_NAME).csr $(CRT_LOCATION)/$(AUTH_GRPC_SERVER_CRT_FILE_NAME).conf
	# Auth gRPC client certificate
	$(file > $(CRT_LOCATION)/$(AUTH_GRPC_CLIENT_CRT_FILE_NAME).conf,$(subst <<SERVICE_NAME>>,$(AUTH_GRPC_CLIENT_CN),$(GRPC_CERT_CONFIG)) )

	openssl req -new -sha256 -newkey rsa:4096 -nodes \
				-keyout $(CRT_LOCATION)/$(AUTH_GRPC_CLIENT_CRT_FILE_NAME).key  \
				-out $(CRT_LOCATION)/$(AUTH_GRPC_CLIENT_CRT_FILE_NAME).csr \
				-config $(CRT_LOCATION)/$(AUTH_GRPC_CLIENT_CRT_FILE_NAME).conf \
				-extensions v3_req

	openssl x509 -req -sha256 \
			-in $(CRT_LOCATION)/$(AUTH_GRPC_CLIENT_CRT_FILE_NAME).csr \
			-CA $(CRT_LOCATION)/ca.crt \
			-CAkey $(CRT_LOCATION)/ca.key \
			-CAcreateserial \
			-out $(CRT_LOCATION)/$(AUTH_GRPC_CLIENT_CRT_FILE_NAME).crt \
			-days 365 \
			-extfile $(CRT_LOCATION)/$(AUTH_GRPC_CLIENT_CRT_FILE_NAME).conf \
			-extensions v3_req

	rm -rf  $(CRT_LOCATION)/$(AUTH_GRPC_CLIENT_CRT_FILE_NAME).csr $(CRT_LOCATION)/$(AUTH_GRPC_CLIENT_CRT_FILE_NAME).conf

agent_certs: agent_ca agent_server_cert agent_client_cert

agent_ca:
	# Create agent directory if it doesn't exist
	mkdir -p $(AGENT_CRT_LOCATION)
	# Generate Agent CA certificate and key
	openssl req -newkey rsa:2048 -x509 -nodes -sha512 -days 1095 \
				-keyout $(AGENT_CRT_LOCATION)/ca.key -out $(AGENT_CRT_LOCATION)/ca.crt \
				-subj "/CN=$(AGENT_CA_CN)/O=$(O)/OU=$(OU_CA)/emailAddress=$(EA)"

agent_server_cert:
	# Create agent server config with SAN
	$(file > $(AGENT_CRT_LOCATION)/agent-server.conf,$(subst <<SERVICE_NAME>>,$(AGENT_SERVER_CN),$(AGENT_CERT_CONFIG)) )
	# Create agent server key and CSR
	openssl req -new -sha256 -newkey rsa:4096 -nodes \
				-keyout $(AGENT_CRT_LOCATION)/agent-key.pem \
				-out $(AGENT_CRT_LOCATION)/agent-server.csr \
				-config $(AGENT_CRT_LOCATION)/agent-server.conf \
				-extensions v3_req
	# Sign agent server CSR with agent CA
	openssl x509 -req -days 730 \
				-in $(AGENT_CRT_LOCATION)/agent-server.csr \
				-CA $(AGENT_CRT_LOCATION)/ca.crt \
				-CAkey $(AGENT_CRT_LOCATION)/ca.key \
				-CAcreateserial \
				-out $(AGENT_CRT_LOCATION)/agent-cert.pem \
				-extfile $(AGENT_CRT_LOCATION)/agent-server.conf \
				-extensions v3_req
	# Remove CSR and config
	rm -rf $(AGENT_CRT_LOCATION)/agent-server.csr $(AGENT_CRT_LOCATION)/agent-server.conf

agent_client_cert:
	# Create agent client key and CSR
	openssl req -new -sha256 -newkey rsa:4096 -nodes \
				-keyout $(AGENT_CRT_LOCATION)/client-key.pem \
				-out $(AGENT_CRT_LOCATION)/agent-client.csr \
				-subj "/CN=$(AGENT_CLIENT_CN)/O=$(O)/OU=$(OU_CRT)/emailAddress=$(EA)"
	# Sign agent client CSR with agent CA
	openssl x509 -req -days 730 \
				-in $(AGENT_CRT_LOCATION)/agent-client.csr \
				-CA $(AGENT_CRT_LOCATION)/ca.crt \
				-CAkey $(AGENT_CRT_LOCATION)/ca.key \
				-CAcreateserial \
				-out $(AGENT_CRT_LOCATION)/client-cert.pem
	# Remove CSR
	rm $(AGENT_CRT_LOCATION)/agent-client.csr
	
clean_certs:
	rm -r $(CRT_LOCATION)/*.crt
	rm -r $(CRT_LOCATION)/*.key
