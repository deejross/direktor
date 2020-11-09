import HomePage from "./components/pages/home/index.js";

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
        fetchDomains: function () {
            const url = "/v1/config/domains";
            fetch(url)
                .then(resp => resp.json())
                .then(data => {
                    if (data.error) {
                        state.error = data.error;
                        return;
                    }

                    state.domains = data.domains;
                })
        }
    },
    mounted: function () {
        this.setColorScheme();
        this.fetchDomains();
    }
});
