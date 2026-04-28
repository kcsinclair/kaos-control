import Event from '../Event.js';
/**
 *
 */
export default class CustomEvent extends Event {
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.detail = eventInit?.detail ?? null;
    }
    /**
     * Init event.
     *
     * @deprecated
     * @param type Type.
     * @param [bubbles=false] "true" if it bubbles.
     * @param [cancelable=false] "true" if it cancelable.
     * @param [detail=null] Custom event detail.
     */
    initCustomEvent(type, bubbles = false, cancelable = false, detail = null) {
        this.type = type;
        this.bubbles = bubbles;
        this.cancelable = cancelable;
        this.detail = detail;
    }
}
//# sourceMappingURL=CustomEvent.js.map