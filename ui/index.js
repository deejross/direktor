import HomePage from "./components/pages/home/index.js";
import SearchPage from "./components/pages/search/index.js";
import AboutPage from "./components/pages/about/index.js";

const state = {
    error: undefined,
    domains: [],
};

window.app = new Vue({
    el: '#app',
    data: state,
    router: new VueRouter({
        routes: [
            {
                path: "/",
                component: HomePage,
                props: () => state
            },
            {
                path: "/search",
                component: SearchPage,
                props: () => state
            },
            {
                path: "/about",
                component: AboutPage,
                props: () => state
            }
        ]
    }),
    vuetify: new Vuetify({

    }),
    methods: {
        setColorScheme: function () {
            const colorSchemeDark = window.matchMedia('(prefers-color-scheme: dark)');
            colorSchemeDark.addEventListener('onchange', () => {
                this.$vuetify.theme.dark = window.matchMedia('(prefers-color-scheme: dark)').matches;
            })

            this.$vuetify.theme.dark = colorSchemeDark.matches;
        },
        loadConfig: function () {
            // TODO: load from localStorage
        }
    },
    mounted: function () {
        this.setColorScheme();
        this.loadConfig();
    }
});
