var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var _Permissions_permissionStatus;
import PermissionStatus from './PermissionStatus.js';
import PermissionNameEnum from './PermissionNameEnum.js';
/**
 * Permissions API.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/Permissions.
 */
class Permissions {
    constructor() {
        _Permissions_permissionStatus.set(this, {});
    }
    /**
     * Returns scroll restoration.
     *
     * @param permissionDescriptor Permission descriptor.
     * @param permissionDescriptor.name Permission name.
     * @param [permissionDescriptor.userVisibleOnly] User visible only.
     * @param [permissionDescriptor.sysex] Sysex.
     * @returns Permission status.
     */
    async query(permissionDescriptor) {
        if (__classPrivateFieldGet(this, _Permissions_permissionStatus, "f")[permissionDescriptor.name]) {
            return __classPrivateFieldGet(this, _Permissions_permissionStatus, "f")[permissionDescriptor.name];
        }
        if (!Object.values(PermissionNameEnum).includes(permissionDescriptor.name)) {
            throw new Error(`Failed to execute 'query' on 'Permissions': Failed to read the 'name' property from 'PermissionDescriptor': The provided value '${permissionDescriptor.name}' is not a valid enum value of type PermissionName.`);
        }
        __classPrivateFieldGet(this, _Permissions_permissionStatus, "f")[permissionDescriptor.name] = new PermissionStatus('granted');
        return __classPrivateFieldGet(this, _Permissions_permissionStatus, "f")[permissionDescriptor.name];
    }
}
_Permissions_permissionStatus = new WeakMap();
export default Permissions;
//# sourceMappingURL=Permissions.js.map