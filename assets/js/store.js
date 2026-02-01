document.addEventListener('alpine:init', () => {
    Alpine.store('global', {
        user: {
            name: '',
            id: '',
            role: ''
        },
        notifications: [],

        init() {
            console.log('Global Store Initialized');
        },

        setUser(user) {
            this.user = user;
        },

        addNotification(message, type = 'info') {
            const id = Date.now();
            this.notifications.push({ id, message, type });
            setTimeout(() => {
                this.removeNotification(id);
            }, 3000);
        },

        removeNotification(id) {
            this.notifications = this.notifications.filter(n => n.id !== id);
        }
    });

    Alpine.store('cart', {
        items: [],

        add(product) {
            this.items.push(product);
        },

        remove(index) {
            this.items.splice(index, 1);
        },

        get total() {
            return this.items.reduce((sum, item) => sum + item.price, 0);
        }
    });
});
