# AxMQ - Next Generation High-Performance MQTT Protocol Full Node

AxMQ is a modern, lightweight, and fully compliant MQTT 5.0/ 3.1.1 / 3.0 protocol implementation designed for 
exceptional performance and scalability. Written in Go, AxMQ provides both server (broker) and client functionality 
with a strong focus on low latency, minimal resource usage, and robust reliability.

## Key Features

- Full MQTT 5.0 and backward-compatible 3.1.1 support  
- Zero-allocation, high-throughput packet parsing and encoding  
- Advanced session, subscription, and QoS management (0, 1, 2)  
- Efficient topic routing with wildcard and shared subscription support  
- Flexible and extensible plugin system for authentication, authorization, and monitoring  
- Built-in clustering and high availability features for enterprise-grade deployments  
- Secure by design with TLS 1.3, certificate-based auth, and fine-grained access control  
- Clear, type-safe API designed for robust integration and custom extensions  
