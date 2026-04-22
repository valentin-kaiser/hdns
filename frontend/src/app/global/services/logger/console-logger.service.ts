import { Injectable } from "@angular/core";
import { environment } from "../../../../environments/environment";
import { Logger, LogLevel } from "./logger.service";

/**
 * To use this service as logger, the service must be set in app.module.ts with useClass
 * providers: [ { provide: LoggerService, useClass: ConsoleLoggerService } ]
 *
 */

/**
 * disable logging in production mode
 */
export let isLoggingEnabled = environment.production ? false : true;

const noop = (): any => undefined;

@Injectable()
export class ConsoleLoggerService implements Logger {
	/**
	 *Initial LogLevel
	 *
	 * @type {LogLevel}
	 * @memberof ConsoleLoggerService
	 */
	_logLevel: LogLevel = LogLevel.DEBUG;

	/**
	 * Bind error log method
	 *
	 * @readonly
	 * @memberof ConsoleLoggerService
	 */
	get error() {
		if (!isLoggingEnabled || this._logLevel < LogLevel.ERROR) {
			return noop;
		}
		return console.error.bind(console);
	}

	/**
	 * Bind warn log method
	 *
	 * @readonly
	 * @memberof ConsoleLoggerService
	 */
	get warn() {
		if (!isLoggingEnabled || this._logLevel < LogLevel.WARN) {
			return noop;
		}
		return console.warn.bind(console);
	}

	/**
	 * Bind notice log method
	 *
	 * @readonly
	 * @memberof ConsoleLoggerService
	 */
	get notice() {
		if (!isLoggingEnabled || this._logLevel < LogLevel.INFO) {
			return noop;
		}
		return console.log.bind(console);
	}

	/**
	 * Bind info log method
	 *
	 * @readonly
	 * @memberof ConsoleLoggerService
	 */
	get info() {
		if (!isLoggingEnabled || this._logLevel < LogLevel.INFO) {
			return noop;
		}
		return console.log.bind(console);
	}

	/**
	 * Bind debug log method
	 *
	 * @readonly
	 * @memberof ConsoleLoggerService
	 */
	get debug() {
		if (!isLoggingEnabled || this._logLevel < LogLevel.DEBUG) {
			return noop;
		}
		return console.log.bind(console);
	}

	/**
	 * Set Log Level
	 *
	 * @param {LogLevel} logLevel
	 * @memberof ConsoleLoggerService
	 */
	setLogLevel(logLevel: LogLevel): void {
		console.warn(`Change logLevel from ${this._logLevel} to ${logLevel}`);
		this._logLevel = logLevel;
	}

	/**
	 * Invoke browser console method
	 *
	 * @param {string} type
	 * @param {*} [args]
	 * @memberof ConsoleLoggerService
	 */
	invokeConsoleMethod(type: string, args?: any): void {
		// tslint:disable-next-line: ban-types
		const logFn: Function = console[type] || console.log || noop;
		logFn.apply(console, [args]);
	}
}
