from src.data_handling import use_item_db, reset_user_limit
from src.items.Item import Item


class Clover(Item):
    def __init__(self):
        super().__init__(
            name="trèfle",
            price=200,
            description="Augmente ta chance. Tu as 75% de chance de réussir ton pile ou face."
        )

    async def use(self, ctx, **kwargs):
        if not use_item_db(ctx.author.id, self.name):
            await ctx.send("❌ Tu n'as pas cet objet.")
            return False
        reset_user_limit(ctx.author.id, "daily")
        await ctx.send(f"🍀 **Zouuuuuu...** La chance est avec toi ! Passe ta probabilité de réussir ton pile ou face à 75%.")
        return True