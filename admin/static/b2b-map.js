/* B2B map: Google Maps (preferred) or Leaflet fallback, plus Bogotá
 * localidad fills and barrio outlines. */
(function (global) {
    "use strict";

    var URBAN = { lat: 4.653, lng: -74.083, zoom: 12 };
    var GEO_LOC = "/admin/static/geo/bogota-localidades.json";
    var GEO_BAR = "/admin/static/geo/bogota-barrios.json";
    var GEO_IDX = "/admin/static/geo/bogota-index.json";

    function escapeHtml(s) {
        return (s || "").replace(/[&<>"']/g, function (c) {
            return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c];
        });
    }

    function pointInRing(lat, lng, ring) {
        var inside = false;
        for (var i = 0, j = ring.length - 1; i < ring.length; j = i++) {
            var xi = ring[i][0], yi = ring[i][1];
            var xj = ring[j][0], yj = ring[j][1];
            var intersect = ((yi > lat) !== (yj > lat)) &&
                (lng < (xj - xi) * (lat - yi) / ((yj - yi) || 1e-12) + xi);
            if (intersect) inside = !inside;
        }
        return inside;
    }

    function pointInGeom(lat, lng, geom) {
        if (!geom) return false;
        if (geom.type === "Polygon") {
            if (!pointInRing(lat, lng, geom.coordinates[0])) return false;
            for (var h = 1; h < geom.coordinates.length; h++) {
                if (pointInRing(lat, lng, geom.coordinates[h])) return false;
            }
            return true;
        }
        if (geom.type === "MultiPolygon") {
            return geom.coordinates.some(function (poly) {
                if (!pointInRing(lat, lng, poly[0])) return false;
                for (var k = 1; k < poly.length; k++) {
                    if (pointInRing(lat, lng, poly[k])) return false;
                }
                return true;
            });
        }
        return false;
    }

    function geomBounds(geom, acc) {
        acc = acc || { south: 90, north: -90, west: 180, east: -180 };
        function walk(coords) {
            if (typeof coords[0] === "number") {
                var lng = coords[0], lat = coords[1];
                if (lat < acc.south) acc.south = lat;
                if (lat > acc.north) acc.north = lat;
                if (lng < acc.west) acc.west = lng;
                if (lng > acc.east) acc.east = lng;
                return;
            }
            coords.forEach(walk);
        }
        walk(geom.coordinates);
        return acc;
    }

    function parseZoneGeom(z) {
        var g = z && (z.Geometry || z.geometry);
        if (!g) return null;
        if (typeof g === "string") {
            try { g = JSON.parse(g); } catch (e) { return null; }
        }
        if (g.type === "Feature") g = g.geometry;
        if (!g || (g.type !== "Polygon" && g.type !== "MultiPolygon")) return null;
        return g;
    }

    function zoneFeatureCollection(list) {
        return {
            type: "FeatureCollection",
            features: (list || []).map(function (z) {
                var geom = parseZoneGeom(z);
                if (!geom) return null;
                return {
                    type: "Feature",
                    properties: {
                        id: z.ID,
                        nombre: z.Name,
                        city: z.City,
                        color: z.Color || "#2563eb"
                    },
                    geometry: geom
                };
            }).filter(Boolean)
        };
    }

    function closeRing(coords) {
        if (!coords.length) return coords;
        var first = coords[0], last = coords[coords.length - 1];
        if (first[0] !== last[0] || first[1] !== last[1]) {
            coords = coords.concat([[first[0], first[1]]]);
        }
        return coords;
    }

    function hasGoogleDrawing() {
        return global.google && google.maps && google.maps.drawing && google.maps.drawing.DrawingManager;
    }

    function ringFromGooglePolygon(polygon) {
        var path = polygon.getPath();
        var coords = [];
        for (var i = 0; i < path.getLength(); i++) {
            var p = path.getAt(i);
            coords.push([p.lng(), p.lat()]);
        }
        return closeRing(coords);
    }

    function fetchJSON(url) {
        return fetch(url, { credentials: "same-origin" }).then(function (r) {
            if (!r.ok) throw new Error("failed " + url);
            return r.json();
        });
    }

    function fillSectorSelects(index, locSel, barSel) {
        if (!locSel || !index) return;
        locSel.innerHTML = '<option value="">Todas las localidades</option>';
        (index.localidades || []).forEach(function (loc) {
            if (loc.codigo === "20") return; // Sumapaz: rural, no barrios
            var opt = document.createElement("option");
            opt.value = loc.nombre;
            opt.textContent = loc.nombre;
            locSel.appendChild(opt);
        });
        if (barSel) {
            barSel.innerHTML = '<option value="">Todos los barrios</option>';
            barSel.disabled = true;
        }
    }

    function populateBarrios(index, locName, barSel) {
        if (!barSel) return;
        barSel.innerHTML = '<option value="">Todos los barrios</option>';
        var loc = (index.localidades || []).find(function (l) { return l.nombre === locName; });
        var list = loc ? loc.barrios || [] : [];
        barSel.disabled = !locName || list.length === 0;
        list.forEach(function (name) {
            var opt = document.createElement("option");
            opt.value = name;
            opt.textContent = name;
            barSel.appendChild(opt);
        });
    }

    function isBogotaCity(name) {
        if (!name) return false;
        var n = String(name).toLowerCase()
            .replace(/á/g, "a").replace(/à/g, "a")
            .replace(/é/g, "e").replace(/í/g, "i")
            .replace(/ó/g, "o").replace(/ú/g, "u").replace(/ü/g, "u")
            .replace(/ñ/g, "n");
        return n.indexOf("bogota") === 0;
    }

    // Cascaded city → localidad → barrio selects for the search form.
    // Only Bogotá has official locality/barrio lists; other cities disable both.
    function bindLocationSelects(opts) {
        var citySel = opts && opts.citySelect;
        var locSel = opts && opts.locSelect;
        var barSel = opts && opts.barSelect;
        if (!citySel || !locSel || !barSel) return Promise.resolve(null);

        function wire(index) {
            function applyCity() {
                if (!isBogotaCity(citySel.value)) {
                    locSel.innerHTML = '<option value="">No hay localidades</option>';
                    locSel.disabled = true;
                    locSel.value = "";
                    barSel.innerHTML = '<option value="">Todos los barrios</option>';
                    barSel.disabled = true;
                    barSel.value = "";
                    return;
                }
                locSel.disabled = false;
                fillSectorSelects(index, locSel, barSel);
                populateBarrios(index, locSel.value, barSel);
            }

            locSel.addEventListener("change", function () {
                populateBarrios(index, locSel.value, barSel);
            });
            citySel.addEventListener("change", applyCity);
            applyCity();
            return index;
        }

        if (opts.index) {
            return Promise.resolve(wire(opts.index));
        }

        return fetchJSON(GEO_IDX).then(wire);
    }

    function createGoogleEngine(el, api) {
        var map = new google.maps.Map(el, {
            center: URBAN,
            zoom: URBAN.zoom,
            mapTypeId: "roadmap",
            streetViewControl: false,
            fullscreenControl: true,
            mapTypeControl: true,
            clickableIcons: false,
            gestureHandling: "greedy"
        });
        var locLayer = new google.maps.Data({ map: map });
        var barLayer = new google.maps.Data({ map: map });
        var zoneLayer = new google.maps.Data({ map: map });
        var markers = [];
        var info = new google.maps.InfoWindow();

        zoneLayer.setStyle(function (feature) {
            var color = feature.getProperty("color") || "#2563eb";
            return {
                fillColor: color,
                fillOpacity: 0.22,
                strokeColor: color,
                strokeWeight: 2.4,
                strokeOpacity: 0.95,
                clickable: true
            };
        });
        zoneLayer.addListener("click", function (e) {
            var name = e.feature.getProperty("nombre") || "Zona";
            var city = e.feature.getProperty("city") || "";
            info.setContent("<strong>" + escapeHtml(name) + "</strong>" +
                (city ? "<div>" + escapeHtml(city) + "</div>" : ""));
            info.setPosition(e.latLng);
            info.open(map);
        });

        locLayer.addListener("click", function (e) {
            if (api.onLocalidadClick) api.onLocalidadClick(e.feature.getProperty("nombre"));
        });
        barLayer.addListener("click", function (e) {
            if (api.onBarrioClick) {
                api.onBarrioClick(e.feature.getProperty("nombre"), e.feature.getProperty("localidad"));
            }
        });

        return {
            kind: "google",
            map: map,
            locLayer: locLayer,
            barLayer: barLayer,
            addGeo: function (layer, fc) { layer.addGeoJson(fc); },
            styleLoc: function (fn) { locLayer.setStyle(fn); },
            styleBar: function (fn) { barLayer.setStyle(fn); },
            setLocVisible: function (on) { locLayer.setMap(on ? map : null); },
            setBarVisible: function (on) { barLayer.setMap(on ? map : null); },
            clearZones: function () {
                var gone = [];
                zoneLayer.forEach(function (f) { gone.push(f); });
                gone.forEach(function (f) { zoneLayer.remove(f); });
            },
            addZones: function (fc) {
                this.clearZones();
                if (fc && fc.features && fc.features.length) {
                    zoneLayer.addGeoJson(fc);
                }
            },
            setZoneVisible: function (on) { zoneLayer.setMap(on ? map : null); },
            clearMarkers: function () {
                markers.forEach(function (m) { m.setMap(null); });
                markers = [];
            },
            addMarker: function (biz, color, onClick) {
                var marker = new google.maps.Marker({
                    position: { lat: biz.lat, lng: biz.lng },
                    map: map,
                    title: biz.title || "",
                    icon: {
                        path: google.maps.SymbolPath.CIRCLE,
                        fillColor: color,
                        fillOpacity: 0.92,
                        strokeColor: "#0b1220",
                        strokeWeight: 1,
                        scale: 8
                    }
                });
                marker.addListener("click", function () { onClick(biz, marker); });
                markers.push(marker);
                return marker;
            },
            styleMarker: function (marker, color) {
                marker.setIcon({
                    path: google.maps.SymbolPath.CIRCLE,
                    fillColor: color,
                    fillOpacity: 0.92,
                    strokeColor: "#0b1220",
                    strokeWeight: 1,
                    scale: 8
                });
            },
            setMarkerVisible: function (marker, on) { marker.setVisible(!!on); },
            fitBounds: function (pts, maxZoom) {
                if (!pts.length) {
                    map.setCenter(URBAN);
                    map.setZoom(URBAN.zoom);
                    return;
                }
                var b = new google.maps.LatLngBounds();
                pts.forEach(function (p) { b.extend(p); });
                map.fitBounds(b, 40);
                if (maxZoom) {
                    google.maps.event.addListenerOnce(map, "idle", function () {
                        if (map.getZoom() > maxZoom) map.setZoom(maxZoom);
                    });
                }
            },
            fitBox: function (box) {
                map.fitBounds({
                    south: box.south, north: box.north, west: box.west, east: box.east
                }, 36);
            },
            openInfo: function (marker, html) {
                info.setContent(html);
                info.open({ map: map, anchor: marker });
            }
        };
    }

    function createLeafletEngine(el) {
        var map = L.map(el, { scrollWheelZoom: true, zoomControl: true }).setView([URBAN.lat, URBAN.lng], URBAN.zoom);
        L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
            maxZoom: 19,
            attribution: "&copy; OpenStreetMap"
        }).addTo(map);
        var locLayer = L.geoJSON(null, { interactive: true }).addTo(map);
        var barLayer = L.geoJSON(null, { interactive: true }).addTo(map);
        var zoneLayer = L.geoJSON(null, {
            interactive: true,
            style: function (feat) {
                var c = (feat.properties && feat.properties.color) || "#2563eb";
                return { color: c, weight: 2.4, fillColor: c, fillOpacity: 0.22, opacity: 0.95 };
            },
            onEachFeature: function (feat, layer) {
                var name = feat.properties && feat.properties.nombre ? feat.properties.nombre : "Zona";
                var city = feat.properties && feat.properties.city ? feat.properties.city : "";
                layer.bindPopup("<strong>" + escapeHtml(name) + "</strong>" +
                    (city ? "<div>" + escapeHtml(city) + "</div>" : ""));
            }
        }).addTo(map);
        var markersLayer = L.layerGroup().addTo(map);
        var markers = [];

        return {
            kind: "leaflet",
            map: map,
            locLayer: locLayer,
            barLayer: barLayer,
            addGeo: function (layer, fc) { layer.addData(fc); },
            styleLoc: function (fn) {
                locLayer.setStyle(function (feat) { return fn(feat.properties || feat); });
            },
            styleBar: function (fn) {
                barLayer.setStyle(function (feat) { return fn(feat.properties || feat); });
            },
            setLocVisible: function (on) {
                if (on) { if (!map.hasLayer(locLayer)) locLayer.addTo(map); }
                else map.removeLayer(locLayer);
            },
            setBarVisible: function (on) {
                if (on) { if (!map.hasLayer(barLayer)) barLayer.addTo(map); }
                else map.removeLayer(barLayer);
            },
            clearZones: function () { zoneLayer.clearLayers(); },
            addZones: function (fc) {
                zoneLayer.clearLayers();
                if (fc && fc.features && fc.features.length) {
                    zoneLayer.addData(fc);
                }
            },
            setZoneVisible: function (on) {
                if (on) { if (!map.hasLayer(zoneLayer)) zoneLayer.addTo(map); }
                else map.removeLayer(zoneLayer);
            },
            clearMarkers: function () {
                markersLayer.clearLayers();
                markers = [];
            },
            addMarker: function (biz, color, onClick) {
                var marker = L.circleMarker([biz.lat, biz.lng], {
                    radius: 7, fillColor: color, color: "#0b1220",
                    weight: 1, opacity: 1, fillOpacity: 0.9
                });
                marker.bindTooltip(escapeHtml(biz.title || "(sin nombre)"));
                marker.on("click", function () { onClick(biz, marker); });
                marker.addTo(markersLayer);
                markers.push(marker);
                return marker;
            },
            styleMarker: function (marker, color) {
                marker.setStyle({ fillColor: color });
            },
            setMarkerVisible: function (marker, on) {
                var el = marker.getElement && marker.getElement();
                if (el) el.style.display = on ? "" : "none";
                if (on) {
                    if (!markersLayer.hasLayer(marker)) marker.addTo(markersLayer);
                } else if (markersLayer.hasLayer(marker)) {
                    markersLayer.removeLayer(marker);
                }
            },
            fitBounds: function (pts, maxZoom) {
                if (!pts.length) {
                    map.setView([URBAN.lat, URBAN.lng], URBAN.zoom);
                    return;
                }
                map.fitBounds(pts, { padding: [30, 30], maxZoom: maxZoom || 14 });
            },
            fitBox: function (box) {
                map.fitBounds([[box.south, box.west], [box.north, box.east]], { padding: [28, 28] });
            },
            openInfo: function (marker, html) {
                marker.unbindPopup();
                marker.bindPopup(html).openPopup();
            }
        };
    }

    function styleLocalidad(p, filterLoc, show) {
        var active = !filterLoc || p.nombre === filterLoc;
        return {
            fillColor: p.color || "#2563eb",
            fillOpacity: show ? (active ? 0.28 : 0.05) : 0,
            strokeColor: p.color || "#2563eb",
            color: p.color || "#2563eb",
            strokeWeight: active ? 2.2 : 0.6,
            weight: active ? 2.2 : 0.6,
            strokeOpacity: show ? (active ? 0.95 : 0.25) : 0,
            opacity: show ? (active ? 0.95 : 0.25) : 0
        };
    }

    function styleBarrio(p, filterLoc, filterBar, show) {
        var inLoc = !filterLoc || p.localidad === filterLoc;
        var isBar = filterBar && p.nombre === filterBar && inLoc;
        var visible = show && inLoc;
        return {
            fillColor: "#000",
            fillOpacity: 0,
            strokeColor: isBar ? "#111827" : (p.color || "#334155"),
            color: isBar ? "#111827" : (p.color || "#334155"),
            strokeWeight: isBar ? 2.4 : 0.7,
            weight: isBar ? 2.4 : 0.7,
            strokeOpacity: visible ? (isBar ? 1 : 0.55) : 0,
            opacity: visible ? (isBar ? 1 : 0.55) : 0
        };
    }

    function googleStyle(fn) {
        return function (feature) {
            return fn({
                nombre: feature.getProperty("nombre"),
                localidad: feature.getProperty("localidad"),
                color: feature.getProperty("color"),
                codigo: feature.getProperty("codigo")
            });
        };
    }

    function startGoogleDrawing(engine, color, onComplete) {
        var dm = new google.maps.drawing.DrawingManager({
            drawingMode: google.maps.drawing.OverlayType.POLYGON,
            drawingControl: false,
            polygonOptions: {
                fillColor: color,
                fillOpacity: 0.25,
                strokeColor: color,
                strokeWeight: 2.4,
                clickable: false,
                editable: true
            }
        });
        dm.setMap(engine.map);
        var listener = google.maps.event.addListener(dm, "polygoncomplete", function (poly) {
            google.maps.event.removeListener(listener);
            dm.setDrawingMode(null);
            dm.setMap(null);
            var ring = ringFromGooglePolygon(poly);
            if (!ring || ring.length < 4) {
                poly.setMap(null);
                return;
            }
            onComplete({ type: "Polygon", coordinates: [ring] }, function () {
                poly.setMap(null);
            });
        });
        return {
            mode: "google",
            finish: function () { return false; },
            cancel: function () {
                google.maps.event.removeListener(listener);
                dm.setDrawingMode(null);
                dm.setMap(null);
            }
        };
    }

    function startClickDraw(engine, color, onVertex, onComplete) {
        var pts = [];
        var preview = null;
        var vertices = [];
        var listeners = [];

        function cleanupPreview() {
            if (engine.kind === "google") {
                if (preview) preview.setMap(null);
                vertices.forEach(function (m) { m.setMap(null); });
            } else {
                if (preview) engine.map.removeLayer(preview);
                vertices.forEach(function (m) { engine.map.removeLayer(m); });
            }
            preview = null;
            vertices = [];
        }

        function redraw() {
            cleanupPreview();
            if (!pts.length) return;
            if (engine.kind === "google") {
                preview = new google.maps.Polyline({
                    path: pts,
                    strokeColor: color,
                    strokeWeight: 2.6,
                    map: engine.map
                });
                pts.forEach(function (p) {
                    vertices.push(new google.maps.Marker({
                        position: p,
                        map: engine.map,
                        icon: {
                            path: google.maps.SymbolPath.CIRCLE,
                            fillColor: "#fff",
                            fillOpacity: 1,
                            strokeColor: color,
                            strokeWeight: 2,
                            scale: 5
                        }
                    }));
                });
            } else {
                preview = L.polyline(pts.map(function (p) { return [p.lat, p.lng]; }), {
                    color: color, weight: 2.6
                }).addTo(engine.map);
                pts.forEach(function (p) {
                    vertices.push(L.circleMarker([p.lat, p.lng], {
                        radius: 5, color: color, fillColor: "#fff", fillOpacity: 1, weight: 2
                    }).addTo(engine.map));
                });
            }
        }

        function toGeom() {
            if (pts.length < 3) return null;
            return {
                type: "Polygon",
                coordinates: [closeRing(pts.map(function (p) { return [p.lng, p.lat]; }))]
            };
        }

        function unbind() {
            if (engine.kind === "google") {
                listeners.forEach(function (l) { google.maps.event.removeListener(l); });
                engine.map.setOptions({ disableDoubleClickZoom: false });
            } else {
                engine.map.off("click", onLeafClick);
                engine.map.doubleClickZoom.enable();
            }
            listeners = [];
        }

        var finished = false;
        function finish() {
            if (finished) return false;
            var geom = toGeom();
            if (!geom) return false;
            finished = true;
            unbind();
            cleanupPreview();
            var draftCleanup;
            if (engine.kind === "google") {
                var poly = new google.maps.Polygon({
                    paths: pts,
                    fillColor: color,
                    fillOpacity: 0.25,
                    strokeColor: color,
                    strokeWeight: 2.4,
                    map: engine.map,
                    clickable: false
                });
                draftCleanup = function () { poly.setMap(null); };
            } else {
                var lpoly = L.polygon(pts.map(function (p) { return [p.lat, p.lng]; }), {
                    color: color, weight: 2.4, fillColor: color, fillOpacity: 0.25
                }).addTo(engine.map);
                draftCleanup = function () { engine.map.removeLayer(lpoly); };
            }
            onComplete(geom, draftCleanup);
            return true;
        }

        function addPoint(lat, lng) {
            pts.push({ lat: lat, lng: lng });
            redraw();
            if (onVertex) onVertex(pts.length);
        }

        function onLeafClick(e) {
            addPoint(e.latlng.lat, e.latlng.lng);
        }

        if (engine.kind === "google") {
            engine.map.setOptions({ disableDoubleClickZoom: true });
            listeners.push(engine.map.addListener("click", function (e) {
                addPoint(e.latLng.lat(), e.latLng.lng());
            }));
        } else {
            engine.map.doubleClickZoom.disable();
            engine.map.on("click", onLeafClick);
        }

        return {
            mode: "click",
            finish: finish,
            cancel: function () {
                unbind();
                cleanupPreview();
            }
        };
    }

    global.B2BMap = {
        escapeHtml: escapeHtml,
        bindLocationSelects: bindLocationSelects,
        create: function (opts) {
            opts = opts || {};
            var el = opts.el;
            var locSel = opts.localidadSelect;
            var barSel = opts.barrioSelect;
            var showLoc = opts.showLocalidades !== false;
            var showBar = opts.showBarrios !== false;
            var filterLoc = "";
            var filterBar = "";
            var index = null;
            var locFC = null;
            var barFC = null;
            var engine = null;
            var items = [];
            var zoneList = [];
            var showZones = true;
            var drawing = false;
            var drawSession = null;
            var draftCleanup = null;

            function currentEngine() {
                if (opts.apiKey && global.google && google.maps) {
                    return createGoogleEngine(el, {
                        onLocalidadClick: function (n) {
                            if (drawing) return;
                            if (locSel) locSel.value = n;
                            filterLoc = n;
                            filterBar = "";
                            if (barSel) populateBarrios(index, n, barSel);
                            applySectors();
                            if (opts.onLocalidadClick) opts.onLocalidadClick(n);
                        },
                        onBarrioClick: function (b, loc) {
                            if (drawing) return;
                            if (locSel && loc) {
                                locSel.value = loc;
                                filterLoc = loc;
                                populateBarrios(index, loc, barSel);
                            }
                            if (barSel) barSel.value = b;
                            filterBar = b;
                            applySectors();
                            if (opts.onBarrioClick) opts.onBarrioClick(b, loc);
                        }
                    });
                }
                return createLeafletEngine(el);
            }

            function applySectors() {
                if (!engine) return;
                if (engine.kind === "google") {
                    engine.styleLoc(googleStyle(function (p) { return styleLocalidad(p, filterLoc, showLoc); }));
                    engine.styleBar(googleStyle(function (p) { return styleBarrio(p, filterLoc, filterBar, showBar); }));
                } else {
                    engine.styleLoc(function (p) { return styleLocalidad(p, filterLoc, showLoc); });
                    engine.styleBar(function (p) { return styleBarrio(p, filterLoc, filterBar, showBar); });
                }
                engine.setLocVisible(showLoc && !drawing);
                engine.setBarVisible(showBar && !drawing);
                if (engine.setZoneVisible) engine.setZoneVisible(showZones && !drawing);
                applyMarkerFilter();
                zoomToFilter();
            }

            function renderZones() {
                if (!engine || !engine.addZones) return;
                engine.addZones(zoneFeatureCollection(zoneList));
                engine.setZoneVisible(showZones && !drawing);
            }

            function stopDraw(keepDraft) {
                drawing = false;
                if (el && el.classList) el.classList.remove("is-drawing");
                if (drawSession) {
                    drawSession.cancel();
                    drawSession = null;
                }
                if (!keepDraft && draftCleanup) {
                    draftCleanup();
                    draftCleanup = null;
                }
                applySectors();
                renderZones();
            }

            function featureForFilter() {
                if (filterBar && barFC) {
                    return barFC.features.find(function (f) {
                        return f.properties.nombre === filterBar &&
                            (!filterLoc || f.properties.localidad === filterLoc);
                    });
                }
                if (filterLoc && locFC) {
                    return locFC.features.find(function (f) { return f.properties.nombre === filterLoc; });
                }
                return null;
            }

            function zoomToFilter() {
                var feat = featureForFilter();
                if (!feat) return;
                engine.fitBox(geomBounds(feat.geometry));
            }

            function inFilter(biz) {
                var feat = featureForFilter();
                if (!feat) return true;
                return pointInGeom(biz.lat, biz.lng, feat.geometry);
            }

            function applyMarkerFilter() {
                items.forEach(function (it) {
                    engine.setMarkerVisible(it.marker, inFilter(it.biz));
                });
            }

            function wireSelects() {
                if (locSel) {
                    locSel.addEventListener("change", function () {
                        filterLoc = locSel.value;
                        filterBar = "";
                        populateBarrios(index, filterLoc, barSel);
                        applySectors();
                    });
                }
                if (barSel) {
                    barSel.addEventListener("change", function () {
                        filterBar = barSel.value;
                        applySectors();
                    });
                }
            }

            return Promise.all([fetchJSON(GEO_IDX), fetchJSON(GEO_LOC), fetchJSON(GEO_BAR)]).then(function (pack) {
                index = pack[0];
                locFC = pack[1];
                barFC = pack[2];
                fillSectorSelects(index, locSel, barSel);
                engine = currentEngine();
                if (engine.kind === "leaflet") {
                    engine.locLayer.on("click", function (e) {
                        if (drawing) return;
                        var n = e.layer && e.layer.feature && e.layer.feature.properties
                            ? e.layer.feature.properties.nombre : "";
                        if (n && locSel) {
                            locSel.value = n;
                            filterLoc = n;
                            filterBar = "";
                            populateBarrios(index, n, barSel);
                            applySectors();
                        }
                    });
                    engine.barLayer.on("click", function (e) {
                        if (drawing) return;
                        var p = e.layer && e.layer.feature ? e.layer.feature.properties : null;
                        if (!p) return;
                        if (locSel && p.localidad) {
                            locSel.value = p.localidad;
                            filterLoc = p.localidad;
                            populateBarrios(index, p.localidad, barSel);
                        }
                        if (barSel) barSel.value = p.nombre;
                        filterBar = p.nombre;
                        applySectors();
                    });
                }
                engine.addGeo(engine.locLayer, locFC);
                engine.addGeo(engine.barLayer, barFC);
                applySectors();
                wireSelects();
                return {
                    engine: engine,
                    index: index,
                    setLayers: function (locOn, barOn) {
                        showLoc = !!locOn;
                        showBar = !!barOn;
                        applySectors();
                    },
                    setMarkers: function (list, colorFor, onClick) {
                        engine.clearMarkers();
                        items = [];
                        var pts = [];
                        (list || []).forEach(function (biz) {
                            if (!biz.lat || !biz.lng) return;
                            var marker = engine.addMarker(biz, colorFor(biz), onClick);
                            items.push({ biz: biz, marker: marker });
                            pts.push(engine.kind === "google"
                                ? { lat: biz.lat, lng: biz.lng }
                                : [biz.lat, biz.lng]);
                        });
                        applyMarkerFilter();
                        if (!featureForFilter()) {
                            engine.fitBounds(pts, 14);
                        }
                        return items;
                    },
                    recolor: function (colorFor) {
                        items.forEach(function (it) {
                            engine.styleMarker(it.marker, colorFor(it.biz));
                        });
                    },
                    visibleCount: function () {
                        return items.filter(function (it) { return inFilter(it.biz); }).length;
                    },
                    openInfo: function (marker, html) { engine.openInfo(marker, html); },
                    usesGoogle: function () { return engine.kind === "google"; },
                    setZones: function (list) {
                        zoneList = list || [];
                        renderZones();
                    },
                    setZonesVisible: function (on) {
                        showZones = !!on;
                        if (engine && engine.setZoneVisible) engine.setZoneVisible(showZones && !drawing);
                    },
                    startDraw: function (drawOpts) {
                        drawOpts = drawOpts || {};
                        stopDraw(false);
                        drawing = true;
                        if (el && el.classList) el.classList.add("is-drawing");
                        applySectors();
                        var color = drawOpts.color || "#2563eb";
                        var onDone = function (geom, cleanup) {
                            drawing = false;
                            if (el && el.classList) el.classList.remove("is-drawing");
                            drawSession = null;
                            draftCleanup = cleanup;
                            applySectors();
                            if (drawOpts.onComplete) drawOpts.onComplete(geom);
                        };
                        if (engine.kind === "google" && hasGoogleDrawing()) {
                            drawSession = startGoogleDrawing(engine, color, onDone);
                        } else {
                            drawSession = startClickDraw(engine, color, drawOpts.onVertex, onDone);
                        }
                        return drawSession.mode;
                    },
                    finishDraw: function () {
                        return !!(drawSession && drawSession.finish && drawSession.finish());
                    },
                    cancelDraw: function () { stopDraw(false); },
                    isDrawing: function () { return drawing; }
                };
            });
        }
    };
})(window);
