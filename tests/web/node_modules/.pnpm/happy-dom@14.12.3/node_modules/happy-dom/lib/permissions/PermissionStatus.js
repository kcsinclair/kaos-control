import EventTarget from '../event/EventTarget.js';
/**
 * Permission status.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/PermissionStatus
 */
export default class PermissionStatus extends EventTarget {
    /**
     * Constructor.
     *
     * @param [state] State.
     */
    constructor(state = 'granted') {
        super();
        this.onchange = null;
        this.state = state;
    }
}
//# sourceMappingURL=PermissionStatus.js.map