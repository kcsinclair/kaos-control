import Event from './Event.js';
/**
 *
 */
class UIEvent extends Event {
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.layerX = 0;
        this.layerY = 0;
        this.pageX = 0;
        this.pageY = 0;
        this.detail = eventInit?.detail ?? 0;
        this.view = eventInit?.view ?? null;
    }
}
UIEvent.NONE = 0;
UIEvent.CAPTURING_PHASE = 1;
UIEvent.AT_TARGET = 2;
UIEvent.BUBBLING_PHASE = 3;
export default UIEvent;
//# sourceMappingURL=UIEvent.js.map