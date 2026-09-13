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

    function ensurePSStyle() {
        if (document.getElementById("b2b-ps-style")) return;
        var st = document.createElement("style");
        st.id = "b2b-ps-style";
        st.textContent = ".ps-pin{width:36px;height:44px;margin-left:-18px;margin-top:-44px;display:flex;align-items:center;justify-content:center;" +
            "background:linear-gradient(135deg,#1d4ed8 0%,#dc2626 100%);color:#fff;font:700 12px/1 Arial,Helvetica,sans-serif;" +
            "border:1.5px solid #0b1220;border-radius:18px 18px 18px 4px;box-shadow:0 3px 8px rgba(15,23,42,.35);transform:rotate(-45deg);}" +
            ".ps-pin span{transform:rotate(45deg);letter-spacing:.02em;}";
        document.head.appendChild(st);
    }

    function priceSmartIconURL() {
        var svg = '<svg xmlns="http://www.w3.org/2000/svg" width="40" height="44" viewBox="0 0 40 44">' +
            '<defs><linearGradient id="psg" x1="0" y1="0" x2="1" y2="1">' +
            '<stop offset="0%" stop-color="#1d4ed8"/><stop offset="100%" stop-color="#dc2626"/></linearGradient></defs>' +
            '<path d="M20 2.2c7.2 0 13 5.6 13 12.6 0 10.2-13 26.6-13 26.6S7 24.9 7 14.8C7 7.8 12.8 2.2 20 2.2z" fill="url(#psg)" stroke="#0b1220" stroke-width="1.4"/>' +
            '<text x="20" y="20" text-anchor="middle" font-size="11" font-weight="700" fill="#fff" font-family="Arial,Helvetica,sans-serif">PS</text></svg>';
        return "data:image/svg+xml;charset=UTF-8," + encodeURIComponent(svg);
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

    function geomRings(geom) {
        if (!geom) return [];
        if (geom.type === "Polygon") return [geom.coordinates[0]];
        if (geom.type === "MultiPolygon") {
            return geom.coordinates.map(function (poly) { return poly[0]; });
        }
        return [];
    }

    function distMeters(a, b) {
        var R = 6371000;
        var dLat = (b.lat - a.lat) * Math.PI / 180;
        var dLng = (b.lng - a.lng) * Math.PI / 180;
        var s = Math.sin(dLat / 2) * Math.sin(dLat / 2) +
            Math.cos(a.lat * Math.PI / 180) * Math.cos(b.lat * Math.PI / 180) *
            Math.sin(dLng / 2) * Math.sin(dLng / 2);
        return 2 * R * Math.asin(Math.min(1, Math.sqrt(s)));
    }

    function ringCentroid(ring) {
        if (!ring || !ring.length) return null;
        var a = 0, x = 0, y = 0;
        var n = ring.length - 1;
        for (var i = 0; i < n; i++) {
            var x0 = ring[i][0], y0 = ring[i][1];
            var x1 = ring[i + 1][0], y1 = ring[i + 1][1];
            var c = x0 * y1 - x1 * y0;
            a += c;
            x += (x0 + x1) * c;
            y += (y0 + y1) * c;
        }
        a *= 0.5;
        var pt;
        if (Math.abs(a) < 1e-12) {
            pt = { lng: ring[0][0], lat: ring[0][1] };
        } else {
            pt = { lng: x / (6 * a), lat: y / (6 * a) };
        }
        if (!pointInRing(pt.lat, pt.lng, ring)) {
            var box = geomBounds({ type: "Polygon", coordinates: [ring] });
            pt = { lng: (box.west + box.east) / 2, lat: (box.south + box.north) / 2 };
        }
        return pt;
    }

    function perpDist(p, a, b) {
        var x = p[0], y = p[1], x1 = a[0], y1 = a[1], x2 = b[0], y2 = b[1];
        var dx = x2 - x1, dy = y2 - y1;
        var len2 = dx * dx + dy * dy;
        if (len2 < 1e-20) return Math.hypot(x - x1, y - y1);
        var t = ((x - x1) * dx + (y - y1) * dy) / len2;
        t = Math.max(0, Math.min(1, t));
        return Math.hypot(x - (x1 + t * dx), y - (y1 + t * dy));
    }

    function simplifyOpen(pts, eps) {
        if (pts.length <= 2) return pts;
        var maxD = 0, idx = 0;
        var last = pts.length - 1;
        for (var i = 1; i < last; i++) {
            var d = perpDist(pts[i], pts[0], pts[last]);
            if (d > maxD) { maxD = d; idx = i; }
        }
        if (maxD > eps) {
            var left = simplifyOpen(pts.slice(0, idx + 1), eps);
            var right = simplifyOpen(pts.slice(idx), eps);
            return left.slice(0, -1).concat(right);
        }
        return [pts[0], pts[last]];
    }

    function simplifyRing(ring, eps) {
        if (!ring || ring.length < 8) return ring;
        var closed = ring[0][0] === ring[ring.length - 1][0] &&
            ring[0][1] === ring[ring.length - 1][1];
        var pts = closed ? ring.slice(0, -1) : ring.slice();
        return closeRing(simplifyOpen(pts, eps || 0.00004));
    }

    var MUTED_ZONE = "#6b7280";

    function ensureHatchPattern(color) {
        var safe = String(color || MUTED_ZONE).replace(/[^#a-fA-F0-9]/g, "") || "6b7280";
        var id = "b2b-hatch-" + safe.replace("#", "");
        if (document.getElementById(id)) return id;
        var svg = document.getElementById("b2b-zone-patterns");
        if (!svg) {
            svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
            svg.setAttribute("id", "b2b-zone-patterns");
            svg.setAttribute("width", "0");
            svg.setAttribute("height", "0");
            svg.style.position = "absolute";
            svg.style.width = "0";
            svg.style.height = "0";
            svg.innerHTML = "<defs></defs>";
            document.body.appendChild(svg);
        }
        var defs = svg.querySelector("defs");
        var pat = document.createElementNS("http://www.w3.org/2000/svg", "pattern");
        pat.setAttribute("id", id);
        pat.setAttribute("patternUnits", "userSpaceOnUse");
        pat.setAttribute("width", "10");
        pat.setAttribute("height", "10");
        var bg = document.createElementNS("http://www.w3.org/2000/svg", "rect");
        bg.setAttribute("width", "10");
        bg.setAttribute("height", "10");
        bg.setAttribute("fill", color);
        bg.setAttribute("opacity", "0.16");
        var dots = document.createElementNS("http://www.w3.org/2000/svg", "circle");
        dots.setAttribute("cx", "2.2");
        dots.setAttribute("cy", "2.2");
        dots.setAttribute("r", "1.25");
        dots.setAttribute("fill", color);
        dots.setAttribute("opacity", "0.95");
        var line = document.createElementNS("http://www.w3.org/2000/svg", "path");
        line.setAttribute("d", "M0 10 L10 0");
        line.setAttribute("stroke", color);
        line.setAttribute("stroke-width", "1.4");
        line.setAttribute("opacity", "0.85");
        pat.appendChild(bg);
        pat.appendChild(dots);
        pat.appendChild(line);
        defs.appendChild(pat);
        return id;
    }

    function zoneInk(props, locVisible) {
        if (locVisible) return MUTED_ZONE;
        return (props && props.color) || "#0891b2";
    }

    function osrmNearest(lat, lng) {
        var url = "https://router.project-osrm.org/nearest/v1/driving/" +
            lng + "," + lat + "?number=1";
        return fetch(url).then(function (r) {
            if (!r.ok) throw new Error("nearest");
            return r.json();
        }).then(function (data) {
            var wp = data.waypoints && data.waypoints[0];
            if (!wp || !wp.location) throw new Error("nearest");
            var snapped = { lng: wp.location[0], lat: wp.location[1] };
            if (typeof wp.distance === "number" && wp.distance > 140) {
                return { lat: lat, lng: lng };
            }
            return snapped;
        });
    }

    function osrmRoute(a, b) {
        var url = "https://router.project-osrm.org/route/v1/driving/" +
            a.lng + "," + a.lat + ";" + b.lng + "," + b.lat +
            "?overview=full&geometries=geojson&continue_straight=true";
        return fetch(url).then(function (r) {
            if (!r.ok) throw new Error("route");
            return r.json();
        }).then(function (data) {
            var coords = data.routes && data.routes[0] && data.routes[0].geometry &&
                data.routes[0].geometry.coordinates;
            if (!coords || coords.length < 2) throw new Error("route");
            return coords.map(function (c) { return { lng: c[0], lat: c[1] }; });
        });
    }

    function googleRoute(a, b) {
        return new Promise(function (resolve, reject) {
            if (!(global.google && google.maps && google.maps.DirectionsService)) {
                reject(new Error("no directions"));
                return;
            }
            var ds = new google.maps.DirectionsService();
            ds.route({
                origin: { lat: a.lat, lng: a.lng },
                destination: { lat: b.lat, lng: b.lng },
                travelMode: google.maps.TravelMode.DRIVING,
                region: "CO",
                provideRouteAlternatives: false
            }, function (res, status) {
                if (status !== "OK" || !res || !res.routes || !res.routes[0]) {
                    reject(new Error(status || "route"));
                    return;
                }
                resolve(res.routes[0].overview_path.map(function (p) {
                    return { lat: p.lat(), lng: p.lng() };
                }));
            });
        });
    }

    function googleSnap(lat, lng) {
        return new Promise(function (resolve, reject) {
            if (!(global.google && google.maps && google.maps.Geocoder)) {
                reject(new Error("no geocoder"));
                return;
            }
            var g = new google.maps.Geocoder();
            g.geocode({ location: { lat: lat, lng: lng } }, function (results, status) {
                if (status !== "OK" || !results || !results[0]) {
                    reject(new Error(status || "geocode"));
                    return;
                }
                var loc = results[0].geometry.location;
                var snapped = { lat: loc.lat(), lng: loc.lng() };
                if (distMeters({ lat: lat, lng: lng }, snapped) > 140) {
                    resolve({ lat: lat, lng: lng });
                    return;
                }
                resolve(snapped);
            });
        });
    }

    function snapToStreet(lat, lng) {
        return osrmNearest(lat, lng).catch(function () {
            return googleSnap(lat, lng);
        }).catch(function () {
            return { lat: lat, lng: lng };
        });
    }

    function routeAlongStreet(a, b, preferGoogle) {
        if (distMeters(a, b) < 8) return Promise.resolve([a, b]);
        var first = preferGoogle ? googleRoute(a, b) : osrmRoute(a, b);
        var fallback = preferGoogle ? function () { return osrmRoute(a, b); } : function () { return googleRoute(a, b); };
        return first.catch(fallback).catch(function () { return [a, b]; });
    }

    function createGoogleZoneChrome(map) {
        var overlay = new google.maps.OverlayView();
        overlay._fc = { type: "FeatureCollection", features: [] };
        overlay._locVisible = true;
        overlay._visible = true;
        overlay.onAdd = function () {
            this.wrap = document.createElement("div");
            this.wrap.className = "b2b-zone-overlay";
            this.wrap.style.cssText = "position:absolute;left:0;top:0;pointer-events:none;";
            this.svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
            this.svg.style.cssText = "position:absolute;overflow:visible;pointer-events:none;";
            this.wrap.appendChild(this.svg);
            this.labels = document.createElement("div");
            this.labels.style.cssText = "position:absolute;left:0;top:0;pointer-events:none;";
            this.wrap.appendChild(this.labels);
            this.getPanes().overlayLayer.appendChild(this.wrap);
        };
        overlay.draw = function () {
            var proj = this.getProjection();
            if (!proj || !this.wrap) return;
            this.wrap.style.display = this._visible ? "" : "none";
            if (!this._visible) return;
            var defs = "";
            var paths = "";
            var labels = "";
            (this._fc.features || []).forEach(function (f) {
                var color = zoneInk(f.properties, overlay._locVisible);
                var pid = ensureHatchPattern(color);
                geomRings(f.geometry).forEach(function (ring, ri) {
                    var d = ring.map(function (c, idx) {
                        var p = proj.fromLatLngToDivPixel(new google.maps.LatLng(c[1], c[0]));
                        return (idx ? "L" : "M") + p.x.toFixed(1) + " " + p.y.toFixed(1);
                    }).join(" ") + " Z";
                    paths += '<path d="' + d + '" fill="url(#' + pid + ')" fill-opacity="0.92" stroke="' +
                        color + '" stroke-width="2.2" stroke-opacity="0.95"></path>';
                    if (ri === 0) {
                        var c = ringCentroid(ring);
                        if (c) {
                            var lp = proj.fromLatLngToDivPixel(new google.maps.LatLng(c.lat, c.lng));
                            var cls = overlay._locVisible ? "is-muted" : "is-color";
                            labels += '<div class="b2b-zone-label ' + cls + '" style="left:' +
                                lp.x.toFixed(1) + 'px;top:' + lp.y.toFixed(1) + 'px;position:absolute;">' +
                                escapeHtml((f.properties && f.properties.nombre) || "Zona") + "</div>";
                        }
                    }
                });
            });
            this.svg.innerHTML = defs + paths;
            this.labels.innerHTML = labels;
        };
        overlay.onRemove = function () {
            if (this.wrap && this.wrap.parentNode) this.wrap.parentNode.removeChild(this.wrap);
        };
        overlay.setFeatures = function (fc) {
            this._fc = fc || { type: "FeatureCollection", features: [] };
            if (this.getProjection()) this.draw();
        };
        overlay.setLocVisible = function (on) {
            this._locVisible = !!on;
            if (this.getProjection()) this.draw();
        };
        overlay.setVisible = function (on) {
            this._visible = !!on;
            if (this.wrap) this.wrap.style.display = on ? "" : "none";
            if (on && this.getProjection()) this.draw();
        };
        overlay.setMap(map);
        return overlay;
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
        var zoneChrome = createGoogleZoneChrome(map);
        var zoneLocVisible = true;
        var markers = [];
        var poiMarkers = [];
        var info = new google.maps.InfoWindow();

        function applyZoneHitStyle() {
            zoneLayer.setStyle(function (feature) {
                var color = zoneInk({ color: feature.getProperty("color") }, zoneLocVisible);
                return {
                    fillColor: color,
                    fillOpacity: 0,
                    strokeColor: color,
                    strokeWeight: 0,
                    strokeOpacity: 0,
                    clickable: false
                };
            });
        }
        applyZoneHitStyle();

        return {
            kind: "google",
            map: map,
            locLayer: locLayer,
            barLayer: barLayer,
            addGeo: function (layer, fc) { layer.addGeoJson(fc); },
            styleLoc: function (fn) {
                locLayer.setStyle(function (feature) {
                    var s = fn(feature);
                    s.clickable = false;
                    return s;
                });
            },
            styleBar: function (fn) {
                barLayer.setStyle(function (feature) {
                    var s = fn(feature);
                    s.clickable = false;
                    return s;
                });
            },
            setLocVisible: function (on) { locLayer.setMap(on ? map : null); },
            setBarVisible: function (on) { barLayer.setMap(on ? map : null); },
            setClickThrough: function () {
                applyZoneHitStyle();
            },
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
                zoneChrome.setFeatures(fc);
                applyZoneHitStyle();
            },
            setZoneVisible: function (on) {
                zoneLayer.setMap(on ? map : null);
                zoneChrome.setVisible(on);
            },
            setZoneDecor: function (opts) {
                zoneLocVisible = !!(opts && opts.locVisible);
                zoneChrome.setLocVisible(zoneLocVisible);
                applyZoneHitStyle();
            },
            clearMarkers: function () {
                markers.forEach(function (m) { m.setMap(null); });
                markers = [];
            },
            clearPOIs: function () {
                poiMarkers.forEach(function (m) { m.setMap(null); });
                poiMarkers = [];
            },
            addPOI: function (poi, onClick) {
                var marker = new google.maps.Marker({
                    position: { lat: poi.lat, lng: poi.lng },
                    map: map,
                    title: poi.name || poi.title || "PriceSmart",
                    zIndex: 2500,
                    icon: {
                        url: priceSmartIconURL(),
                        scaledSize: new google.maps.Size(40, 44),
                        anchor: new google.maps.Point(20, 42)
                    }
                });
                marker.addListener("click", function () { onClick(poi, marker); });
                poiMarkers.push(marker);
                return marker;
            },
            addMarker: function (biz, color, onClick) {
                var marker = new google.maps.Marker({
                    position: { lat: biz.lat, lng: biz.lng },
                    map: map,
                    title: biz.title || "",
                    zIndex: 1000,
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
        map.createPane("locPane");
        map.getPane("locPane").style.zIndex = 350;
        map.getPane("locPane").style.pointerEvents = "none";
        map.createPane("barPane");
        map.getPane("barPane").style.zIndex = 360;
        map.getPane("barPane").style.pointerEvents = "none";
        map.createPane("zonePane");
        map.getPane("zonePane").style.zIndex = 370;
        map.getPane("zonePane").style.pointerEvents = "none";
        map.createPane("zoneLabelPane");
        map.getPane("zoneLabelPane").style.zIndex = 380;
        map.getPane("zoneLabelPane").style.pointerEvents = "none";
        map.createPane("bizPane");
        map.getPane("bizPane").style.zIndex = 650;
        map.createPane("poiPane");
        map.getPane("poiPane").style.zIndex = 680;

        var locLayer = L.geoJSON(null, { interactive: false, pane: "locPane" }).addTo(map);
        var barLayer = L.geoJSON(null, { interactive: false, pane: "barPane" }).addTo(map);
        var zoneLocVisible = true;
        var zoneLabelLayer = L.layerGroup({ pane: "zoneLabelPane" }).addTo(map);
        var zoneLayer = L.geoJSON(null, {
            interactive: false,
            pane: "zonePane",
            style: function (feat) {
                var c = zoneInk(feat.properties, zoneLocVisible);
                return { color: c, weight: 2.2, fillColor: c, fillOpacity: 0.1, opacity: 0.95 };
            }
        }).addTo(map);
        var markersLayer = L.layerGroup({ pane: "bizPane" }).addTo(map);
        var poiLayer = L.layerGroup({ pane: "poiPane" }).addTo(map);
        var markers = [];
        var poiMarkers = [];

        function paintLeafletZones() {
            zoneLayer.eachLayer(function (layer) {
                var feat = layer.feature || {};
                var color = zoneInk(feat.properties, zoneLocVisible);
                var pid = ensureHatchPattern(color);
                layer.setStyle({
                    color: color, weight: 2.2, fillColor: color, fillOpacity: 0.12, opacity: 0.95
                });
                if (layer._path) {
                    layer._path.setAttribute("fill", "url(#" + pid + ")");
                    layer._path.setAttribute("fill-opacity", "0.92");
                }
            });
            zoneLabelLayer.clearLayers();
            zoneLayer.eachLayer(function (layer) {
                var feat = layer.feature;
                if (!feat) return;
                var ring = geomRings(feat.geometry)[0];
                var c = ringCentroid(ring);
                if (!c) return;
                var name = (feat.properties && feat.properties.nombre) || "Zona";
                var icon = L.divIcon({
                    className: "b2b-zone-label " + (zoneLocVisible ? "is-muted" : "is-color"),
                    html: "<span>" + escapeHtml(name) + "</span>",
                    iconSize: [0, 0],
                    iconAnchor: [0, 0]
                });
                L.marker([c.lat, c.lng], { icon: icon, interactive: false, pane: "zoneLabelPane" })
                    .addTo(zoneLabelLayer);
            });
        }
        map.on("zoomend moveend", paintLeafletZones);

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
            setClickThrough: function () {
                ["locPane", "barPane", "zonePane"].forEach(function (name) {
                    var pane = map.getPane(name);
                    if (pane) pane.style.pointerEvents = "none";
                });
            },
            clearZones: function () {
                zoneLayer.clearLayers();
                zoneLabelLayer.clearLayers();
            },
            addZones: function (fc) {
                zoneLayer.clearLayers();
                zoneLabelLayer.clearLayers();
                if (fc && fc.features && fc.features.length) {
                    zoneLayer.addData(fc);
                }
                paintLeafletZones();
            },
            setZoneVisible: function (on) {
                if (on) {
                    if (!map.hasLayer(zoneLayer)) zoneLayer.addTo(map);
                    if (!map.hasLayer(zoneLabelLayer)) zoneLabelLayer.addTo(map);
                } else {
                    map.removeLayer(zoneLayer);
                    map.removeLayer(zoneLabelLayer);
                }
            },
            setZoneDecor: function (opts) {
                zoneLocVisible = !!(opts && opts.locVisible);
                paintLeafletZones();
            },
            clearMarkers: function () {
                markersLayer.clearLayers();
                markers = [];
            },
            clearPOIs: function () {
                poiLayer.clearLayers();
                poiMarkers = [];
            },
            addPOI: function (poi, onClick) {
                ensurePSStyle();
                var icon = L.divIcon({
                    className: "",
                    html: '<div class="ps-pin"><span>PS</span></div>',
                    iconSize: [36, 44],
                    iconAnchor: [18, 44]
                });
                var marker = L.marker([poi.lat, poi.lng], { icon: icon, pane: "poiPane", zIndexOffset: 800 });
                marker.bindTooltip(escapeHtml(poi.name || poi.title || "PriceSmart"));
                marker.on("click", function () { onClick(poi, marker); });
                marker.addTo(poiLayer);
                poiMarkers.push(marker);
                return marker;
            },
            addMarker: function (biz, color, onClick) {
                var marker = L.circleMarker([biz.lat, biz.lng], {
                    radius: 7, fillColor: color, color: "#0b1220",
                    weight: 1, opacity: 1, fillOpacity: 0.9,
                    pane: "bizPane", interactive: true
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

    function startStreetDraw(engine, color, onVertex, onComplete) {
        var corners = [];
        var segments = [];
        var preview = null;
        var vertices = [];
        var listeners = [];
        var busy = false;
        var finished = false;
        var preferGoogle = engine.kind === "google";

        function notify() {
            if (onVertex) onVertex(corners.length, { routing: busy });
        }

        function fullPath() {
            var out = [];
            segments.forEach(function (seg, i) {
                var start = i === 0 ? 0 : 1;
                for (var k = start; k < seg.length; k++) out.push(seg[k]);
            });
            return out;
        }

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
            var pts = fullPath();
            if (!pts.length && !corners.length) return;
            if (engine.kind === "google") {
                if (pts.length) {
                    preview = new google.maps.Polyline({
                        path: pts,
                        strokeColor: color,
                        strokeWeight: 3,
                        geodesic: false,
                        map: engine.map
                    });
                }
                corners.forEach(function (p) {
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
                if (pts.length) {
                    preview = L.polyline(pts.map(function (p) { return [p.lat, p.lng]; }), {
                        color: color, weight: 3
                    }).addTo(engine.map);
                }
                corners.forEach(function (p) {
                    vertices.push(L.circleMarker([p.lat, p.lng], {
                        radius: 5, color: color, fillColor: "#fff", fillOpacity: 1, weight: 2
                    }).addTo(engine.map));
                });
            }
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

        function showDraft(path) {
            cleanupPreview();
            var draftCleanup;
            if (engine.kind === "google") {
                var poly = new google.maps.Polygon({
                    paths: path,
                    fillColor: color,
                    fillOpacity: 0.18,
                    strokeColor: color,
                    strokeWeight: 2.6,
                    map: engine.map,
                    clickable: false
                });
                draftCleanup = function () { poly.setMap(null); };
            } else {
                var lpoly = L.polygon(path.map(function (p) { return [p.lat, p.lng]; }), {
                    color: color, weight: 2.6, fillColor: color, fillOpacity: 0.18
                }).addTo(engine.map);
                draftCleanup = function () { engine.map.removeLayer(lpoly); };
            }
            return draftCleanup;
        }

        function finish() {
            if (finished || busy || corners.length < 3) return false;
            finished = true;
            busy = true;
            notify();
            var last = corners[corners.length - 1];
            var first = corners[0];
            routeAlongStreet(last, first, preferGoogle).then(function (seg) {
                if (seg && seg.length >= 2) segments.push(seg);
                var ring = simplifyRing(closeRing(fullPath().map(function (p) {
                    return [p.lng, p.lat];
                })), 0.000035);
                unbind();
                var draft = showDraft(fullPath().concat([first]));
                onComplete({ type: "Polygon", coordinates: [ring] }, draft);
            }).catch(function () {
                finished = false;
                busy = false;
                notify();
            });
            return true;
        }

        function undo() {
            if (busy || finished || !corners.length) return;
            corners.pop();
            segments.pop();
            redraw();
            notify();
        }

        function addPoint(lat, lng) {
            if (busy || finished) return;
            busy = true;
            notify();
            snapToStreet(lat, lng).then(function (pt) {
                if (!corners.length) {
                    corners.push(pt);
                    segments.push([pt]);
                    redraw();
                    return;
                }
                var prev = corners[corners.length - 1];
                return routeAlongStreet(prev, pt, preferGoogle).then(function (seg) {
                    corners.push(pt);
                    segments.push(seg && seg.length ? seg : [prev, pt]);
                    redraw();
                });
            }).catch(function () {
                var raw = { lat: lat, lng: lng };
                if (!corners.length) {
                    corners.push(raw);
                    segments.push([raw]);
                } else {
                    corners.push(raw);
                    segments.push([corners[corners.length - 2], raw]);
                }
                redraw();
            }).then(function () {
                busy = false;
                notify();
            });
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
            undo: undo,
            cancel: function () {
                unbind();
                cleanupPreview();
            }
        };
    }

    global.B2BMap = {
        escapeHtml: escapeHtml,
        pointInGeom: pointInGeom,
        parseZoneGeom: parseZoneGeom,
        bindLocationSelects: bindLocationSelects,
        create: function (opts) {
            opts = opts || {};
            var el = opts.el;
            var locSel = opts.localidadSelect;
            var barSel = opts.barrioSelect;
            var showLoc = opts.showLocalidades !== false;
            var showBar = opts.showBarrios === true;
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
                    return createGoogleEngine(el, {});
                }
                return createLeafletEngine(el);
            }

            function applySectors() {
                if (!engine) return;
                if (engine.setClickThrough) engine.setClickThrough(drawing);
                if (engine.kind === "google") {
                    engine.styleLoc(googleStyle(function (p) { return styleLocalidad(p, filterLoc, showLoc); }));
                    engine.styleBar(googleStyle(function (p) { return styleBarrio(p, filterLoc, filterBar, showBar); }));
                } else {
                    engine.styleLoc(function (p) { return styleLocalidad(p, filterLoc, showLoc); });
                    engine.styleBar(function (p) { return styleBarrio(p, filterLoc, filterBar, showBar); });
                }
                engine.setLocVisible(showLoc && !drawing);
                engine.setBarVisible(showBar && !drawing);
                if (engine.setZoneVisible) engine.setZoneVisible(showZones);
                if (engine.setZoneDecor) engine.setZoneDecor({ locVisible: showLoc });
                applyMarkerFilter();
                zoomToFilter();
            }

            function renderZones() {
                if (!engine || !engine.addZones) return;
                engine.addZones(zoneFeatureCollection(zoneList));
                engine.setZoneVisible(showZones);
                if (engine.setZoneDecor) engine.setZoneDecor({ locVisible: showLoc });
                if (engine.setClickThrough) engine.setClickThrough(drawing);
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
                    engine.setMarkerVisible(it.marker, !drawing && inFilter(it.biz));
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
                    setPOIs: function (list, onClick) {
                        if (engine.clearPOIs) engine.clearPOIs();
                        (list || []).forEach(function (poi) {
                            if (!poi.lat || !poi.lng || !engine.addPOI) return;
                            engine.addPOI(poi, onClick || function () {});
                        });
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
                        if (engine && engine.setZoneVisible) engine.setZoneVisible(showZones);
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
                        drawSession = startStreetDraw(engine, color, drawOpts.onVertex, onDone);
                        return drawSession.mode;
                    },
                    finishDraw: function () {
                        return !!(drawSession && drawSession.finish && drawSession.finish());
                    },
                    undoDraw: function () {
                        if (drawSession && drawSession.undo) drawSession.undo();
                    },
                    cancelDraw: function () { stopDraw(false); },
                    isDrawing: function () { return drawing; }
                };
            });
        }
    };
})(window);
