import * as PropertySymbol from '../PropertySymbol.js';
import Storage from './Storage.js';
/**
 * Dataset factory.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLElement/storage
 */
export default class StorageFactory {
    /**
     * Creates a new storage.
     */
    static createStorage() {
        // Documentation for Proxy:
        // https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Proxy
        return new Proxy(new Storage(), {
            get(storage, key) {
                if (Storage.prototype.hasOwnProperty(key)) {
                    const descriptor = Object.getOwnPropertyDescriptor(Storage.prototype, key);
                    if (descriptor.value !== undefined) {
                        if (typeof descriptor.value === 'function') {
                            return storage[key].bind(storage);
                        }
                        return descriptor.value;
                    }
                    if (descriptor.get) {
                        return descriptor.get.call(storage);
                    }
                    return storage[key];
                }
                return storage[PropertySymbol.data][key];
            },
            set(storage, key, value) {
                if (Storage.prototype.hasOwnProperty(key)) {
                    return true;
                }
                storage[PropertySymbol.data][key] = String(value);
                return true;
            },
            deleteProperty(storage, key) {
                if (Storage.prototype.hasOwnProperty(key)) {
                    return true;
                }
                return delete storage[PropertySymbol.data][key];
            },
            ownKeys(storage) {
                return Object.keys(storage[PropertySymbol.data]);
            },
            has(storage, key) {
                return storage[PropertySymbol.data][key] !== undefined;
            },
            defineProperty(storage, key, descriptor) {
                if (Storage.prototype.hasOwnProperty(key)) {
                    if (descriptor.get || descriptor.set) {
                        Object.defineProperty(storage, key, {
                            ...descriptor,
                            get: descriptor.get ? descriptor.get.bind(storage) : undefined,
                            set: descriptor.set ? descriptor.set.bind(storage) : undefined
                        });
                    }
                    else {
                        Object.defineProperty(storage, key, {
                            ...descriptor,
                            value: typeof descriptor.value === 'function'
                                ? descriptor.value.bind(storage)
                                : descriptor.value
                        });
                    }
                    return true;
                }
                if (descriptor.value === undefined) {
                    return false;
                }
                storage[PropertySymbol.data][key] = String(descriptor.value);
                return true;
            },
            getOwnPropertyDescriptor(storage, key) {
                if (Storage.prototype.hasOwnProperty(key) ||
                    storage[PropertySymbol.data][key] === undefined) {
                    return;
                }
                return {
                    value: storage[PropertySymbol.data][key],
                    writable: true,
                    enumerable: true,
                    configurable: true
                };
            }
        });
    }
}
//# sourceMappingURL=StorageFactory.js.map