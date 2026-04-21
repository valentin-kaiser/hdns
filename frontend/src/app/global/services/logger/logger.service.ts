import { Injectable } from "@angular/core";

const noop = (): any => undefined;

/**
 * Logger methods
 *
 * @export
 * @abstract
 * @class Logger
 */
export abstract class Logger {
	error: any;
	warn: any;
	notice: any;
	info: any;
	debug: any;
}

/**
 * Loglevels
 *
 * @export
 * @enum {number}
 */
export enum LogLevel {
	ERROR = 0,
	WARN,
	NOTICE,
	INFO,
	DEBUG,
}

@Injectable()
export class LoggerService implements Logger {
	error: any;
	warn: any;
	notice: any;
	info: any;
	debug: any;

	invokeConsoleMethod(type: string, args?: any): void { }

	setLogLevel(logLevel: LogLevel): void { }
}
