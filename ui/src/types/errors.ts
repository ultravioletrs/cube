export class HttpError extends Error {
  statusCode: number;
  name: string;
  error?: string;
  constructor(
    name: string,
    statusCode: number,
    message: string,
    error?: string,
  ) {
    super(message);
    this.name = name;
    this.statusCode = statusCode;
    this.error = error;
  }
  toString() {
    return `${this.name}: ${this.error?.toString()}`;
  }
}
