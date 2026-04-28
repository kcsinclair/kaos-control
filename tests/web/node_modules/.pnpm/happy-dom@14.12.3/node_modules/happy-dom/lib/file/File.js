import Blob from './Blob.js';
/**
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/File.
 *
 * Based on:
 * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/file-api/File-impl.js (MIT licensed).
 */
export default class File extends Blob {
    /**
     * Constructor.
     *
     * @param bits File bits.
     * @param name File name.
     * @param [options] Options.
     * @param [options.type] MIME type.
     * @param [options.lastModifier] Last modified. Defaults to Date.now().
     * @param options.lastModified
     */
    constructor(bits, name, options) {
        super(bits, options);
        this.lastModified = null;
        this.name = null;
        this.name = name.replace(/\//g, ':');
        this.lastModified = options && options.lastModified ? options.lastModified : Date.now();
    }
}
//# sourceMappingURL=File.js.map