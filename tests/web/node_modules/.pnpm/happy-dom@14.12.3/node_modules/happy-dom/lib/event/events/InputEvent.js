import UIEvent from '../UIEvent.js';
/**
 *
 */
export default class InputEvent extends UIEvent {
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.data = eventInit?.data ?? '';
        this.dataTransfer = eventInit?.dataTransfer ?? null;
        this.inputType = eventInit?.inputType ?? '';
        this.isComposing = eventInit?.isComposing ?? false;
    }
}
//# sourceMappingURL=InputEvent.js.map