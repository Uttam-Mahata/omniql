/**
 * OmniQL Java binding module.
 *
 * This module wraps the native OmniQL shared library via JNI and exposes a
 * fluent Java API for executing OQL queries against multiple database backends.
 */
module io.omniql {
    requires com.google.gson;
    exports io.omniql;
}
