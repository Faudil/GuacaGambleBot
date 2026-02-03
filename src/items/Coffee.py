from src.data_handling import use_item_db, reset_user_limit
from src.items.Item import Item


class Coffee(Item):
    def __init__(self):
        super().__init__(
            name="Café",
            price=50,
            description="Réveille tes sens. Réinitialise le cooldown du !daily."
        )

    async def use(self, ctx, **kwargs):
        if not use_item_db(ctx.author.id, self.name):
            await ctx.send("❌ Tu n'as pas cet objet.")
            return False

        reset_user_limit(ctx.author.id, "daily")
        await ctx.send("☕ **Gloupgloup...** Tu es réveillé ! Ton `!daily` est de nouveau disponible.")
        return True