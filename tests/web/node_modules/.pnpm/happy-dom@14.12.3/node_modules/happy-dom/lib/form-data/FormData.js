var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var __classPrivateFieldSet = (this && this.__classPrivateFieldSet) || function (receiver, state, value, kind, f) {
    if (kind === "m") throw new TypeError("Private method is not writable");
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
    return (kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value)), value;
};
var _FormData_instances, _FormData_entries, _FormData_parseValue;
import Blob from '../file/Blob.js';
import * as PropertySymbol from '../PropertySymbol.js';
import File from '../file/File.js';
import RadioNodeList from '../nodes/html-form-element/RadioNodeList.js';
const SUBMITTABLE_ELEMENTS = ['BUTTON', 'INPUT', 'OBJECT', 'SELECT', 'TEXTAREA'];
/**
 * FormData.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/FormData
 */
class FormData {
    /**
     * Constructor.
     *
     * @param [form] Form.
     */
    constructor(form) {
        _FormData_instances.add(this);
        _FormData_entries.set(this, []);
        if (form) {
            for (const name of Object.keys(form[PropertySymbol.elements][PropertySymbol.namedItems])) {
                let radioNodeList = form[PropertySymbol.elements][PropertySymbol.namedItems][name];
                if (radioNodeList[0][PropertySymbol.tagName] === 'INPUT' &&
                    (radioNodeList[0].type === 'checkbox' || radioNodeList[0].type === 'radio')) {
                    const newRadioNodeList = new RadioNodeList();
                    for (const node of radioNodeList) {
                        if (node.checked) {
                            newRadioNodeList.push(node);
                            break;
                        }
                    }
                    radioNodeList = newRadioNodeList;
                }
                for (const node of radioNodeList) {
                    if (node.name && SUBMITTABLE_ELEMENTS.includes(node[PropertySymbol.tagName])) {
                        if (node[PropertySymbol.tagName] === 'INPUT' && node.type === 'file') {
                            if (node[PropertySymbol.files].length === 0) {
                                this.append(node.name, new File([], '', { type: 'application/octet-stream' }));
                            }
                            else {
                                for (const file of node[PropertySymbol.files]) {
                                    this.append(node.name, file);
                                }
                            }
                        }
                        else if (node.value) {
                            this.append(node.name, node.value);
                        }
                    }
                }
            }
        }
    }
    /**
     * For each.
     *
     * @param callback Callback.
     */
    forEach(callback) {
        for (const entry of __classPrivateFieldGet(this, _FormData_entries, "f")) {
            callback.call(this, entry.value, entry.name, this);
        }
    }
    /**
     * Appends a new value onto an existing key.
     *
     * @param name Name.
     * @param value Value.
     * @param [filename] Filename.
     */
    append(name, value, filename) {
        __classPrivateFieldGet(this, _FormData_entries, "f").push({
            name,
            value: __classPrivateFieldGet(this, _FormData_instances, "m", _FormData_parseValue).call(this, value, filename)
        });
    }
    /**
     * Removes a value.
     *
     * @param name Name.
     */
    delete(name) {
        const newEntries = [];
        for (const entry of __classPrivateFieldGet(this, _FormData_entries, "f")) {
            if (entry.name !== name) {
                newEntries.push(entry);
            }
        }
        __classPrivateFieldSet(this, _FormData_entries, newEntries, "f");
    }
    /**
     * Returns value.
     *
     * @param name Name.
     * @returns Value.
     */
    get(name) {
        for (const entry of __classPrivateFieldGet(this, _FormData_entries, "f")) {
            if (entry.name === name) {
                return entry.value;
            }
        }
        return null;
    }
    /**
     * Returns all values associated with the given name.
     *
     * @param name Name.
     * @returns Values.
     */
    getAll(name) {
        const values = [];
        for (const entry of __classPrivateFieldGet(this, _FormData_entries, "f")) {
            if (entry.name === name) {
                values.push(entry.value);
            }
        }
        return values;
    }
    /**
     * Returns whether a FormData object contains a certain key.
     *
     * @param name Name.
     * @returns "true" if the FormData object contains the key.
     */
    has(name) {
        for (const entry of __classPrivateFieldGet(this, _FormData_entries, "f")) {
            if (entry.name === name) {
                return true;
            }
        }
        return false;
    }
    /**
     * Sets a new value for an existing key inside a FormData object, or adds the key/value if it does not already exist.
     *
     * @param name Name.
     * @param value Value.
     * @param [filename] Filename.
     */
    set(name, value, filename) {
        for (const entry of __classPrivateFieldGet(this, _FormData_entries, "f")) {
            if (entry.name === name) {
                entry.value = __classPrivateFieldGet(this, _FormData_instances, "m", _FormData_parseValue).call(this, value, filename);
                return;
            }
        }
        this.append(name, value);
    }
    /**
     * Returns an iterator, allowing you to go through all keys of the key/value pairs contained in this object.
     *
     * @returns Iterator.
     */
    *keys() {
        for (const entry of __classPrivateFieldGet(this, _FormData_entries, "f")) {
            yield entry.name;
        }
    }
    /**
     * Returns an iterator, allowing you to go through all values of the key/value pairs contained in this object.
     *
     * @returns Iterator.
     */
    *values() {
        for (const entry of __classPrivateFieldGet(this, _FormData_entries, "f")) {
            yield entry.value;
        }
    }
    /**
     * Returns an iterator, allowing you to go through all key/value pairs contained in this object.
     *
     * @returns Iterator.
     */
    *entries() {
        for (const entry of __classPrivateFieldGet(this, _FormData_entries, "f")) {
            yield [entry.name, entry.value];
        }
    }
    /**
     * Iterator.
     *
     * @returns Iterator.
     */
    *[(_FormData_entries = new WeakMap(), _FormData_instances = new WeakSet(), Symbol.iterator)]() {
        for (const entry of __classPrivateFieldGet(this, _FormData_entries, "f")) {
            yield [entry.name, entry.value];
        }
    }
}
_FormData_parseValue = function _FormData_parseValue(value, filename) {
    if (value instanceof Blob && !(value instanceof File)) {
        const file = new File([], 'blob', { type: value.type });
        file[PropertySymbol.buffer] = value[PropertySymbol.buffer];
        return file;
    }
    if (value instanceof File) {
        if (filename) {
            const file = new File([], filename, { type: value.type, lastModified: value.lastModified });
            file[PropertySymbol.buffer] = value[PropertySymbol.buffer];
            return file;
        }
        return value;
    }
    return String(value);
};
export default FormData;
//# sourceMappingURL=FormData.js.map