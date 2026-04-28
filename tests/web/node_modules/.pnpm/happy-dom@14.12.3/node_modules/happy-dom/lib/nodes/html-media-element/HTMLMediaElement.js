var _a, _b, _c, _d, _e, _f, _g, _h, _j, _k, _l, _m, _o, _p, _q, _r, _s, _t, _u;
import Event from '../../event/Event.js';
import DOMException from '../../exception/DOMException.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import TimeRange from './TimeRange.js';
/**
 * HTML Media Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLMediaElement.
 *
 */
export default class HTMLMediaElement extends HTMLElement {
    constructor() {
        super(...arguments);
        // Events
        this.onabort = null;
        this.oncanplay = null;
        this.oncanplaythrough = null;
        this.ondurationchange = null;
        this.onemptied = null;
        this.onended = null;
        this.onerror = null;
        this.onloadeddata = null;
        this.onloadedmetadata = null;
        this.onloadstart = null;
        this.onpause = null;
        this.onplay = null;
        this.onplaying = null;
        this.onprogress = null;
        this.onratechange = null;
        this.onresize = null;
        this.onseeked = null;
        this.onseeking = null;
        this.onstalled = null;
        this.onsuspend = null;
        this.ontimeupdate = null;
        this.onvolumechange = null;
        this.onwaiting = null;
        // Internal Properties
        this[_a] = 1;
        this[_b] = true;
        this[_c] = 0;
        this[_d] = 1;
        this[_e] = 1;
        this[_f] = false;
        this[_g] = false;
        this[_h] = true;
        this[_j] = new TimeRange();
        this[_k] = NaN;
        this[_l] = null;
        this[_m] = false;
        this[_o] = 0;
        this[_p] = 0;
        this[_q] = [];
        this[_r] = [];
        this[_s] = false;
        this[_t] = new TimeRange();
        this[_u] = new TimeRange();
    }
    /**
     * Returns buffered.
     *
     * @returns Buffered.
     */
    get buffered() {
        return this[PropertySymbol.buffered];
    }
    /**
     * Returns duration.
     *
     * @returns Duration.
     */
    get duration() {
        return this[PropertySymbol.duration];
    }
    /**
     * Returns error.
     *
     * @returns Error.
     */
    get error() {
        return this[PropertySymbol.error];
    }
    /**
     * Returns ended.
     *
     * @returns Ended.
     */
    get ended() {
        return this[PropertySymbol.ended];
    }
    /**
     * Returns networkState.
     *
     * @returns NetworkState.
     */
    get networkState() {
        return this[PropertySymbol.networkState];
    }
    /**
     * Returns readyState.
     *
     * @returns ReadyState.
     */
    get readyState() {
        return this[PropertySymbol.readyState];
    }
    /**
     * Returns textTracks.
     *
     * @returns TextTracks.
     */
    get textTracks() {
        return this[PropertySymbol.textTracks];
    }
    /**
     * Returns videoTracks.
     *
     * @returns VideoTracks.
     */
    get videoTracks() {
        return this[PropertySymbol.videoTracks];
    }
    /**
     * Returns seeking.
     *
     * @returns Seeking.
     */
    get seeking() {
        return this[PropertySymbol.seeking];
    }
    /**
     * Returns seekable.
     *
     * @returns Seekable.
     */
    get seekable() {
        return this[PropertySymbol.seekable];
    }
    /**
     * Returns played.
     *
     * @returns Played.
     */
    get played() {
        return this[PropertySymbol.played];
    }
    /**
     * Returns autoplay.
     *
     * @returns Autoplay.
     */
    get autoplay() {
        return this.getAttribute('autoplay') !== null;
    }
    /**
     * Sets autoplay.
     *
     * @param autoplay Autoplay.
     */
    set autoplay(autoplay) {
        if (!autoplay) {
            this.removeAttribute('autoplay');
        }
        else {
            this.setAttribute('autoplay', '');
        }
    }
    /**
     * Returns controls.
     *
     * @returns Controls.
     */
    get controls() {
        return this.getAttribute('controls') !== null;
    }
    /**
     * Sets controls.
     *
     * @param controls Controls.
     */
    set controls(controls) {
        if (!controls) {
            this.removeAttribute('controls');
        }
        else {
            this.setAttribute('controls', '');
        }
    }
    /**
     * Returns loop.
     *
     * @returns Loop.
     */
    get loop() {
        return this.getAttribute('loop') !== null;
    }
    /**
     * Sets loop.
     *
     * @param loop Loop.
     */
    set loop(loop) {
        if (!loop) {
            this.removeAttribute('loop');
        }
        else {
            this.setAttribute('loop', '');
        }
    }
    /**
     * Returns muted.
     *
     * @returns Muted.
     */
    get muted() {
        if (this[PropertySymbol.muted]) {
            return this[PropertySymbol.muted];
        }
        if (!this[PropertySymbol.defaultMuted]) {
            return this.getAttribute('muted') !== null;
        }
        return false;
    }
    /**
     * Sets muted.
     *
     * @param muted Muted.
     */
    set muted(muted) {
        this[PropertySymbol.muted] = !!muted;
        if (!muted && !this[PropertySymbol.defaultMuted]) {
            this.removeAttribute('muted');
        }
        else {
            this.setAttribute('muted', '');
        }
    }
    /**
     * Returns defaultMuted.
     *
     * @returns DefaultMuted.
     */
    get defaultMuted() {
        return this[PropertySymbol.defaultMuted];
    }
    /**
     * Sets defaultMuted.
     *
     * @param defaultMuted DefaultMuted.
     */
    set defaultMuted(defaultMuted) {
        this[PropertySymbol.defaultMuted] = !!defaultMuted;
        if (!this[PropertySymbol.defaultMuted] && !this[PropertySymbol.muted]) {
            this.removeAttribute('muted');
        }
        else {
            this.setAttribute('muted', '');
        }
    }
    /**
     * Returns src.
     *
     * @returns Src.
     */
    get src() {
        return this.getAttribute('src') || '';
    }
    /**
     * Sets src.
     *
     * @param src Src.
     */
    set src(src) {
        this.setAttribute('src', src);
        if (Boolean(src)) {
            this.dispatchEvent(new Event('canplay', { bubbles: false, cancelable: false }));
            this.dispatchEvent(new Event('durationchange', { bubbles: false, cancelable: false }));
        }
    }
    /**
     * Returns currentSrc.
     *
     * @returns CurrentrSrc.
     */
    get currentSrc() {
        return this.src;
    }
    /**
     * Returns volume.
     *
     * @returns Volume.
     */
    get volume() {
        return this[PropertySymbol.volume];
    }
    /**
     * Sets volume.
     *
     * @param volume Volume.
     */
    set volume(volume) {
        const parsedVolume = Number(volume);
        if (isNaN(parsedVolume)) {
            throw new TypeError(`Failed to set the 'volume' property on 'HTMLMediaElement': The provided double value is non-finite.`);
        }
        if (parsedVolume < 0 || parsedVolume > 1) {
            throw new DOMException(`Failed to set the 'volume' property on 'HTMLMediaElement': The volume provided (${parsedVolume}) is outside the range [0, 1].`, DOMExceptionNameEnum.indexSizeError);
        }
        // TODO: volumechange event https://developer.mozilla.org/en-US/docs/Web/API/HTMLMediaElement/volumechange_event
        this[PropertySymbol.volume] = parsedVolume;
    }
    /**
     * Returns crossOrigin.
     *
     * @returns CrossOrigin.
     */
    get crossOrigin() {
        return this.getAttribute('crossorigin');
    }
    /**
     * Sets crossOrigin.
     *
     * @param crossOrigin CrossOrigin.
     */
    set crossOrigin(crossOrigin) {
        if (crossOrigin === null) {
            return;
        }
        if (['', 'use-credentials', 'anonymous'].includes(crossOrigin)) {
            this.setAttribute('crossorigin', crossOrigin);
        }
        else {
            this.setAttribute('crossorigin', 'anonymous');
        }
    }
    /**
     * Returns currentTime.
     *
     * @returns CurrentTime.
     */
    get currentTime() {
        return this[PropertySymbol.currentTime];
    }
    /**
     * Sets currentTime.
     *
     * @param currentTime CurrentTime.
     */
    set currentTime(currentTime) {
        const parsedCurrentTime = Number(currentTime);
        if (isNaN(parsedCurrentTime)) {
            throw new TypeError(`Failed to set the 'currentTime' property on 'HTMLMediaElement': The provided double value is non-finite.`);
        }
        this[PropertySymbol.currentTime] = parsedCurrentTime;
    }
    /**
     * Returns playbackRate.
     *
     * @returns PlaybackRate.
     */
    get playbackRate() {
        return this[PropertySymbol.playbackRate];
    }
    /**
     * Sets playbackRate.
     *
     * @param playbackRate PlaybackRate.
     */
    set playbackRate(playbackRate) {
        const parsedPlaybackRate = Number(playbackRate);
        if (isNaN(parsedPlaybackRate)) {
            throw new TypeError(`Failed to set the 'playbackRate' property on 'HTMLMediaElement': The provided double value is non-finite.`);
        }
        this[PropertySymbol.playbackRate] = parsedPlaybackRate;
    }
    /**
     * Returns defaultPlaybackRate.
     *
     * @returns DefaultPlaybackRate.
     */
    get defaultPlaybackRate() {
        return this[PropertySymbol.defaultPlaybackRate];
    }
    /**
     * Sets defaultPlaybackRate.
     *
     * @param defaultPlaybackRate DefaultPlaybackRate.
     */
    set defaultPlaybackRate(defaultPlaybackRate) {
        const parsedDefaultPlaybackRate = Number(defaultPlaybackRate);
        if (isNaN(parsedDefaultPlaybackRate)) {
            throw new TypeError(`Failed to set the 'defaultPlaybackRate' property on 'HTMLMediaElement': The provided double value is non-finite.`);
        }
        this[PropertySymbol.defaultPlaybackRate] = parsedDefaultPlaybackRate;
    }
    /**
     * Returns preservesPitch.
     *
     * @returns PlaybackRate.
     */
    get preservesPitch() {
        return this[PropertySymbol.preservesPitch];
    }
    /**
     * Sets preservesPitch.
     *
     * @param preservesPitch PreservesPitch.
     */
    set preservesPitch(preservesPitch) {
        this[PropertySymbol.preservesPitch] = Boolean(preservesPitch);
    }
    /**
     * Returns preload.
     *
     * @returns preload.
     */
    get preload() {
        return this.getAttribute('preload') || 'auto';
    }
    /**
     * Sets preload.
     *
     * @param preload preload.
     */
    set preload(preload) {
        this.setAttribute('preload', preload);
    }
    /**
     * Returns paused.
     *
     * @returns Paused.
     */
    get paused() {
        return this[PropertySymbol.paused];
    }
    /**
     * Pause played media.
     */
    pause() {
        this[PropertySymbol.paused] = true;
        this.dispatchEvent(new Event('pause', { bubbles: false, cancelable: false }));
    }
    /**
     * Start playing media.
     */
    async play() {
        this[PropertySymbol.paused] = false;
        return Promise.resolve();
    }
    /**
     *
     * @param _type
     */
    canPlayType(_type) {
        return '';
    }
    /**
     * Load media.
     */
    load() {
        this.dispatchEvent(new Event('emptied', { bubbles: false, cancelable: false }));
    }
    /**
     *
     */
    captureStream() {
        return {};
    }
    /**
     * @override
     */
    [(_a = PropertySymbol.volume, _b = PropertySymbol.paused, _c = PropertySymbol.currentTime, _d = PropertySymbol.playbackRate, _e = PropertySymbol.defaultPlaybackRate, _f = PropertySymbol.muted, _g = PropertySymbol.defaultMuted, _h = PropertySymbol.preservesPitch, _j = PropertySymbol.buffered, _k = PropertySymbol.duration, _l = PropertySymbol.error, _m = PropertySymbol.ended, _o = PropertySymbol.networkState, _p = PropertySymbol.readyState, _q = PropertySymbol.textTracks, _r = PropertySymbol.videoTracks, _s = PropertySymbol.seeking, _t = PropertySymbol.seekable, _u = PropertySymbol.played, PropertySymbol.cloneNode)](deep = false) {
        return super[PropertySymbol.cloneNode](deep);
    }
}
//# sourceMappingURL=HTMLMediaElement.js.map