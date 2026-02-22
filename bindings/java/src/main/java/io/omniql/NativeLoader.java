package io.omniql;

import java.io.File;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;

/**
 * NativeLoader handles extracting and loading the native shared library
 * from the JAR file.
 */
class NativeLoader {
    private static boolean loaded = false;

    static synchronized void load() {
        if (loaded) return;

        String os = System.getProperty("os.name").toLowerCase();
        String arch = System.getProperty("os.arch").toLowerCase();
        
        String prefix = "lib";
        String suffix = ".so";
        
        if (os.contains("win")) {
            prefix = "";
            suffix = ".dll";
        } else if (os.contains("mac")) {
            suffix = ".dylib";
        }

        String libName = prefix + "omniql" + suffix;
        String resourcePath = "/native/" + libName;

        try (InputStream is = OmniEngine.class.getResourceAsStream(resourcePath)) {
            if (is == null) {
                // Fallback to system library path if not in JAR
                System.loadLibrary("omniql");
                loaded = true;
                return;
            }

            File tempFile = File.createTempFile("libomniql-", suffix);
            tempFile.deleteOnExit();

            try (FileOutputStream osStream = new FileOutputStream(tempFile)) {
                byte[] buffer = new byte[8192];
                int read;
                while ((read = is.read(buffer)) != -1) {
                    osStream.write(buffer, 0, read);
                }
            }

            System.load(tempFile.getAbsolutePath());
            loaded = true;
        } catch (IOException e) {
            throw new RuntimeException("Failed to load native library: " + libName, e);
        } catch (UnsatisfiedLinkError e) {
            // Last resort: try system load
            System.loadLibrary("omniql");
            loaded = true;
        }
    }
}
