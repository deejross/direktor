export default {
    props: [
        "domains"
    ],
    methods: {

    },
    template: `
        <v-navigation-drawer>
            <v-list-item>
                <v-list-item-content>
                    <v-list-item-title>Domains</v-list-item-title>
                </v-list-item-content>
            </v-list-item>

            <v-list dense nav>
                <v-list-item v-for="(domain, index) of domains" :key="index">
                    {{ domain.name }}
                </v-list-item>
            </v-list>      
        </v-navigation-drawer>
    `
}