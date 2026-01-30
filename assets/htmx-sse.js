/* assets/htmx-sse.js */
(function () {
    'use strict';
    var api;

    htmx.defineExtension("sse", {
        init: function (apiRef) {
            api = apiRef;
        },
        onEvent: function (name, evt) {
            var element = evt.detail.elt;

            // 1. Khởi tạo kết nối (htmx:beforeProcessNode)
            if (name === "htmx:beforeProcessNode") {
                var sseConnect = api.getAttributeValue(element, "sse-connect");
                if (sseConnect) {
                    // Sử dụng withCredentials để gửi kèm cookie auth nếu cần
                    var eventSource = new EventSource(sseConnect, { withCredentials: true });

                    // --- Error Handling ---
                    eventSource.onerror = function (err) {
                        api.triggerEvent(element, "htmx:sseError", { error: err, source: eventSource });
                        console.warn("[SSE] Connection unstable/lost. Retrying...", err);
                    };

                    // --- Connection Open ---
                    eventSource.onopen = function (evt) {
                        api.triggerEvent(element, "htmx:sseOpen", { source: eventSource });
                        console.log("[SSE] Connected to " + sseConnect);
                    };

                    // Lưu instance vào internal data của HTMX
                    api.getInternalData(element).sseEventSource = eventSource;

                    // --- Handle Swaps (sse-swap) ---
                    var sseSwap = api.getAttributeValue(element, "sse-swap");
                    if (sseSwap) {
                        var swapEvents = sseSwap.split(',');
                        swapEvents.forEach(function (eventName) {
                            var cleanName = eventName.trim();

                            eventSource.addEventListener(cleanName, function (event) {
                                // [NEW] Trigger event để Alpine/JS bên ngoài có thể xử lý Data trước khi Swap
                                var allowSwap = api.triggerEvent(element, "htmx:sseMessage", {
                                    event: cleanName,
                                    data: event.data
                                });

                                if (!allowSwap) return; // Nếu preventDefault() thì không swap HTML

                                // Xử lý Swap HTML tiêu chuẩn
                                var swapSpec = api.getSwapSpecification(element);
                                var target = api.getTarget(element);
                                var settlement = api.makeSettleInfo(target);

                                api.swap(target, event.data, swapSpec);

                                settlement.elts.forEach(function (elt) {
                                    if (elt.classList) elt.classList.add(htmx.config.addedClass);
                                    api.triggerEvent(elt, "htmx:load");
                                });
                            });
                        });
                    }

                    // [NEW] Server yêu cầu đóng kết nối (ví dụ: hết phiên làm việc)
                    eventSource.addEventListener("sse-close", function () {
                        console.log("[SSE] Closing connection by server request");
                        eventSource.close();
                    });
                }
            }

            // 2. Dọn dẹp (htmx:beforeCleanupElement)
            if (name === "htmx:beforeCleanupElement") {
                var internalData = api.getInternalData(element);
                if (internalData.sseEventSource) {
                    internalData.sseEventSource.close();
                    internalData.sseEventSource = null; // Tránh memory leak
                }
            }
        }
    });
})();