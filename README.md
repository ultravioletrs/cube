# Cube AI

Cube AI is a framework for building GPT-based AI applications using confidential computing. It protects user data and the AI model by using a trusted execution environment (TEE). TEE is a secure area of a processor that ensures that code and data loaded inside it are protected with respect to confidentiality and integrity. Data confidentiality prevents unauthorized access of data from outside the TEE, while code integrity ensures that code inside the TEE remains unchanged and unaltered from unauthorized access.

## Key Features

- **Secure Computing**: Cube AI uses secure enclaves to protect user data and AI models from unauthorized access.
- **Trusted Execution Environment (TEE)**: Cube AI uses a trusted execution environment to ensure that AI models are executed securely and in a controlled environment.
- **Scalability**: Cube AI can handle large amounts of data and AI models, making it suitable for applications that require high performance and scalability.

## Why Cube AI?

Traditional GPT-based AI applications often rely on public cloud services, which can be vulnerable to security breaches and unauthorized access. The tenant for example openai, and the hardware provider for example Azure, are not always transparent about their security practices and can be easily compromised. They can also access your prompts and model responses. Cube AI addresses these privacy concerns by using TEEs. Using TEEs, Cube AI ensures that user data and AI models are protected from unauthorized access outside the TEE. This helps to maintain user privacy and ensures that AI models are used in a controlled and secure manner.

## How does Cube AI work?

Cube AI uses TEEs to protect user data and AI models from unauthorized access. TEE offers an execution space that provides a higher level of security for trusted applications running on the device. In Cube AI, the TEE ensures that AI models are executed securely and in a controlled environment.

## Documentation

Project documentation is hosted at [Cube AI docs repository](https://github.com/ultravioletrs/cube-docs).

## License

Cocos AI is published under permissive open-source [Apache-2.0](LICENSE) license.
